/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consorsbank

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const Name string = "consorsbank"

type API struct {
	address                   string
	tlsConfig                 *tls.Config
	secret                    string
	preferredCurrency         finance.Currency
	preferredExchanges        []string
	subscriptionTimeout       time.Duration
	session                   *apiSession
	exchangeRateSubscriptions map[string]*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate]
	quoteSubscriptions        map[string]*streamSubscription[proto.SecurityMarketDataReply, finance.Quote]
	logger                    *slog.Logger
	mutex                     sync.Mutex
	stoppedWG                 sync.WaitGroup
}

type apiSession struct {
	grpcClient  *grpc.ClientConn
	AccessToken string
}

func (s *apiSession) SecurityService() proto.SecurityServiceClient {
	return proto.NewSecurityServiceClient(s.grpcClient)
}

func NewAPI(config Config) (*API, error) {
	logger := slog.With(slog.String("provider", Name))
	address, err := config.GetAddress()
	if err != nil {
		return nil, err
	}
	if address == "" {
		address = DefaultAddress
	}
	tlsConfig, err := config.GetTLSConfig()
	if err != nil {
		return nil, err
	}
	secret, err := config.GetSecret()
	if err != nil {
		return nil, err
	}
	preferredCurrency, err := config.GetPreferredCurrency()
	if err != nil {
		return nil, err
	}
	preferredExchanges, err := config.GetPreferredExchanges()
	if err != nil {
		return nil, err
	}
	if len(preferredExchanges) == 0 {
		preferredExchanges = []string{DefaultExchange}
	}
	subscriptionTimeout, err := config.GetSubscriptionTimeout()
	if err != nil {
		return nil, err
	}
	api := &API{
		address:                   address,
		tlsConfig:                 tlsConfig,
		secret:                    secret,
		preferredCurrency:         preferredCurrency,
		preferredExchanges:        preferredExchanges,
		subscriptionTimeout:       subscriptionTimeout,
		exchangeRateSubscriptions: make(map[string]*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate]),
		quoteSubscriptions:        make(map[string]*streamSubscription[proto.SecurityMarketDataReply, finance.Quote]),
		logger:                    logger,
	}
	return api, nil
}

func (api *API) Shutdown(ctx context.Context) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	return errors.Join(api.shutdownSessionLocked(ctx), api.closeSessionLocked())
}

func (api *API) Close() error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	return api.closeSessionLocked()
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) getSecurityInfo(ctx context.Context, session *apiSession, securityService proto.SecurityServiceClient, securityCode *proto.SecurityCode) (*proto.SecurityInfoReply, error) {
	reply, err := securityService.GetSecurityInfo(ctx, &proto.SecurityInfoRequest{
		AccessToken:  session.AccessToken,
		SecurityCode: securityCode,
	})
	if err != nil {
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("failed to query security info from Consorsbank TAPI (cause: %w)", err)
	}
	if tapiError := reply.GetError(); tapiError != nil {
		return nil, fmt.Errorf("query security info error in Consorsbank TAPI call (code: %s, message: %s)",
			tapiError.GetCode(), tapiError.GetMessage())
	}
	return reply, nil
}

func (api *API) getSecurityMarketDataSubscriptionLocked(instrument *finance.Instrument) *streamSubscription[proto.SecurityMarketDataReply, finance.Quote] {
	subscriptionKey, _ := instrument.Identifier(finance.InstrumentIdentifierISIN)
	subscription := api.quoteSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startSecurityMarketDataSubscriptionLocked(ctx context.Context, instrument *finance.Instrument) (*streamSubscription[proto.SecurityMarketDataReply, finance.Quote], error) {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	subscriptionKey, _ := instrument.Identifier(finance.InstrumentIdentifierISIN)
	securityCode := &proto.SecurityCode{
		Code:     subscriptionKey,
		CodeType: proto.SecurityCodeType_ISIN,
	}
	securityInfoReply, err := api.getSecurityInfo(ctx, session, securityService, securityCode)
	if err != nil {
		return nil, err
	}
	if len(securityInfoReply.StockExchangeInfos) == 0 {
		return nil, fmt.Errorf("unable to determine stock exchange for quote query (symbol: %s)", securityCode.Code)
	}
	preferredExchange, err := api.getPreferredExchange(subscriptionKey, securityInfoReply.StockExchangeInfos)
	if err != nil {
		return nil, err
	}
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := securityService.StreamMarketData(subscriptionCtx, &proto.SecurityMarketDataRequest{
		AccessToken: session.AccessToken,
		SecurityWithStockexchange: &proto.SecurityWithStockExchange{
			SecurityCode:  securityCode,
			StockExchange: preferredExchange.StockExchange,
		},
		Currency: string(api.preferredCurrency),
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("failed to create security market data subscription (cause: %w)", err)
	}
	subscription := &streamSubscription[proto.SecurityMarketDataReply, finance.Quote]{
		client:              client,
		subscriptionTimeout: api.subscriptionTimeout,
		recordReply: func(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], reply *proto.SecurityMarketDataReply) {
			recordSecurityMarketDataReply(s, instrument, reply)
		},
		ctx:    subscriptionCtx,
		cancel: subscriptionCancel,
		logger: api.logger.With(slog.String("securityMarketDataSubscription", subscriptionKey)),
	}
	api.quoteSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
}

func (api *API) getPreferredExchange(isin string, exchangeInfos []*proto.SecurityStockExchangeInfo) (*proto.SecurityStockExchangeInfo, error) {
	for _, preferredExchange := range api.preferredExchanges {
		for _, exchangeInfo := range exchangeInfos {
			if exchangeInfo.StockExchange.Id == preferredExchange {
				return exchangeInfo, nil
			}
		}
	}
	return nil, fmt.Errorf("unable to determine preferred exchange for ISIN: %s", isin)
}

func (api *API) getSessionLocked(ctx context.Context) (*apiSession, error) {
	if api.session != nil {
		return api.session, nil
	}
	api.logger.Info("creating Consorsbank TAPI session...")
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(api.tlsConfig)),
	}
	grpcClient, err := grpc.NewClient(api.address, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client to Consorsbank TAPI at '%s' (cause: %w)", api.address, err)
	}
	accessService := proto.NewAccessServiceClient(grpcClient)
	loginReply, err := accessService.Login(ctx, &proto.LoginRequest{Secret: api.secret})
	if err != nil {
		grpcClient.Close()
		return nil, fmt.Errorf("failed to login to Consorsbank TAPI (cause: %w)", err)
	}
	api.session = &apiSession{
		grpcClient:  grpcClient,
		AccessToken: loginReply.GetAccessToken(),
	}
	return api.session, nil
}

func (api *API) invalidateSessionLocked() {
	api.session = nil
}

func (api *API) shutdownSessionLocked(ctx context.Context) error {
	for _, exchangeRateSubscription := range api.exchangeRateSubscriptions {
		exchangeRateSubscription.cancel()
	}
	for _, quoteSubscription := range api.quoteSubscriptions {
		quoteSubscription.cancel()
	}
	api.stoppedWG.Wait()
	if api.session == nil {
		return nil
	}
	accessService := proto.NewAccessServiceClient(api.session.grpcClient)
	_, err := accessService.Logout(ctx, &proto.LogoutRequest{AccessToken: api.session.AccessToken})
	if err != nil {
		return fmt.Errorf("failed to send logout request to Consorsbank TAPI (cause: %w)", err)
	}
	return nil
}

func (api *API) closeSessionLocked() error {
	if api.session == nil {
		return nil
	}
	err := api.session.grpcClient.Close()
	api.session = nil
	if err != nil {
		return fmt.Errorf("failed to close Consorsbank TAPI gRPC client (cause: %w)", err)
	}
	return nil
}

type streamSubscription[R, T any] struct {
	client              grpc.ServerStreamingClient[R]
	subscriptionTimeout time.Duration
	suscribeUntil       time.Time
	lastReply           *T
	recordReply         func(*streamSubscription[R, T], *R)
	ctx                 context.Context
	cancel              context.CancelFunc
	closed              bool
	logger              *slog.Logger
	mutex               sync.Mutex
}

func (s *streamSubscription[R, T]) IsClosed() bool {
	if s == nil {
		return true
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.closed
}

func (s *streamSubscription[R, T]) markClosed() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.closed = true
}

func (s *streamSubscription[R, T]) Run() {
	s.logger.Info("subscription running...")
	for {
		if s.timeoutReached() {
			s.logger.Info("closing subscription; timeout reached")
			s.markClosed()
			return
		}
		reply, err := s.client.Recv()
		if err != nil {
			s.markClosed()
			if errors.Is(err, io.EOF) {
				s.logger.Info("closing subscription; server closed connection")
			} else if s.ctx.Err() != nil || s.client.Context().Err() != nil {
				s.logger.Info("closing subscription; client is disconnecting")
			} else {
				s.logger.Info("closing subscription; recv failure", slog.Any("err", err))
			}
			return
		}
		s.recordReply(s, reply)
	}
}

func (s *streamSubscription[R, T]) timeoutReached() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.subscriptionTimeout == 0 {
		return false
	}
	now := time.Now()
	if s.suscribeUntil.IsZero() {
		s.suscribeUntil = now.Add(s.subscriptionTimeout)
		return false
	}
	return now.After(s.suscribeUntil)
}

func (s *streamSubscription[R, T]) LastReply() (*T, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.lastReply == nil {
		return nil, finance.ErrRequestPending
	}
	// Only start extending timeout after at least one reply has come
	// to timeout disfunctional subscriptions.
	s.suscribeUntil = time.Now().Add(s.subscriptionTimeout)
	return s.lastReply, nil
}

func recordCurrencyRateReply(s *streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate], reply *proto.CurrencyRateReply) {
	if reply.Error != nil {
		s.logger.Debug("ignoring errornous currency rate reply", slog.Any("err", reply.Error))
		return
	}
	if reply.CurrencyFrom == "" || reply.CurrencyTo == "" || math.IsNaN(reply.CurrencyRate) {
		s.logger.Debug("ignoring empty currency rate reply")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("recording currency rate reply")
	s.lastReply = currencyRateReplyToExchangeRate(reply)
}

func recordSecurityMarketDataReply(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], instrument *finance.Instrument, reply *proto.SecurityMarketDataReply) {
	if reply.Error != nil {
		s.logger.Debug("ignoring errornous market data reply", slog.Any("err", reply.Error))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("recording security market data reply")
	s.lastReply = securityMarketDataReplyToQuote(instrument, reply)
}
