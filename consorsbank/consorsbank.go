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
	"strings"
	"sync"

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
	tlsConfig, err := config.GetTLSConfig()
	if err != nil {
		return nil, err
	}
	secret, err := config.GetSecret()
	if err != nil {
		return nil, err
	}
	api := &API{
		address:                   address,
		tlsConfig:                 tlsConfig,
		secret:                    secret,
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

func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	subscription := api.getExchangeRateSubscriptionLocked(base, quote)
	if subscription == nil {
		var err error
		subscription, err = api.startExchangeRateSubscriptionLocked(ctx, base, quote)
		if err != nil {
			return nil, err
		}
	}
	return subscription.LastReply()
}

func (api *API) getExchangeRateSubscriptionLocked(base, quote finance.Currency) *streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate] {
	subscriptionKey := fmt.Sprintf("%s/%s", base, quote)
	subscription := api.exchangeRateSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startExchangeRateSubscriptionLocked(ctx context.Context, base, quote finance.Currency) (*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate], error) {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	subscriptionKey := fmt.Sprintf("%s/%s", base, quote)
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := securityService.StreamCurrencyRate(subscriptionCtx, &proto.CurrencyRateRequest{
		AccessToken:  session.AccessToken,
		CurrencyFrom: string(base),
		CurrencyTo:   string(quote),
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("failed to create exchange rate subscription (cause: %w)", err)
	}
	subscription := &streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate]{
		client:      client,
		recordReply: recordCurrencyRateReply,
		ctx:         subscriptionCtx,
		cancel:      subscriptionCancel,
		logger:      api.logger.With(slog.String("currencyRateSubscription", subscriptionKey)),
	}
	api.exchangeRateSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
}

func (api *API) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	querySecurityCodes := api.resolveQuerySecurityCodes(query)
	if len(querySecurityCodes) == 0 {
		return nil, finance.ErrSymbolSearchRestricted
	}
	symbols := make(finance.Symbols, 0, len(querySecurityCodes))
	for _, querySecurityCode := range querySecurityCodes {
		reply, err := api.getSecurityInfo(ctx, session, securityService, querySecurityCode)
		if err != nil {
			return nil, err
		}
		symbol := securityInfoToSymbol(reply)
		if !symbol.IsEmpty() {
			symbols = append(symbols, *symbol)
		}
	}
	if len(symbols) == 0 {
		return nil, finance.ErrSymbolNotAvailable
	}
	return symbols, nil
}

func (api *API) resolveQuerySecurityCodes(query string) []*proto.SecurityCode {
	securityCodes := make([]*proto.SecurityCode, 0)
	queryFields := strings.Fields(query)
	for _, queryField := range queryFields {
		if finance.IsISIN(queryField) {
			securityCodes = append(securityCodes, &proto.SecurityCode{
				Code:     queryField,
				CodeType: proto.SecurityCodeType_ISIN,
			})
		} else if finance.IsWKN(queryField) {
			securityCodes = append(securityCodes, &proto.SecurityCode{
				Code:     queryField,
				CodeType: proto.SecurityCodeType_WKN,
			})
		}
	}
	return securityCodes
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

func (api *API) QueryQuote(ctx context.Context, symbol finance.Symbol) (*finance.Quote, error) {
	if !symbol.HasISIN() {
		return nil, finance.ErrQuoteNotAvailable
	}

	api.mutex.Lock()
	defer api.mutex.Unlock()

	subscription := api.getSecurityMarketDataSubscriptionLocked(&symbol)
	if subscription == nil {
		var err error
		subscription, err = api.startSecurityMarketDataSubscriptionLocked(ctx, &symbol)
		if err != nil {
			return nil, err
		}
	}
	return subscription.LastReply()
}

func (api *API) getSecurityMarketDataSubscriptionLocked(symbol *finance.Symbol) *streamSubscription[proto.SecurityMarketDataReply, finance.Quote] {
	subscriptionKey := symbol.ISIN
	subscription := api.quoteSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startSecurityMarketDataSubscriptionLocked(ctx context.Context, symbol *finance.Symbol) (*streamSubscription[proto.SecurityMarketDataReply, finance.Quote], error) {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCode := &proto.SecurityCode{
		Code:     symbol.ISIN,
		CodeType: proto.SecurityCodeType_ISIN,
	}
	securityInfoReply, err := api.getSecurityInfo(ctx, session, securityService, securityCode)
	if err != nil {
		return nil, err
	}
	if len(securityInfoReply.StockExchangeInfos) == 0 {
		return nil, fmt.Errorf("unable to determine stock exchange for quote query (symbol: %s)", securityCode.Code)
	}
	subscriptionKey := symbol.ISIN
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := securityService.StreamMarketData(subscriptionCtx, &proto.SecurityMarketDataRequest{
		AccessToken: session.AccessToken,
		SecurityWithStockexchange: &proto.SecurityWithStockExchange{
			SecurityCode:  securityCode,
			StockExchange: securityInfoReply.StockExchangeInfos[0].StockExchange,
		},
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("failed to create security market data subscription (cause: %w)", err)
	}
	subscription := &streamSubscription[proto.SecurityMarketDataReply, finance.Quote]{
		client: client,
		recordReply: func(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], reply *proto.SecurityMarketDataReply) {
			recordSecurityMarketDataReply(s, symbol, reply)
		},
		ctx:    subscriptionCtx,
		cancel: subscriptionCancel,
		logger: api.logger.With(slog.String("securityMarketDataSubscription", subscriptionKey)),
	}
	api.quoteSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
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
	if api.session == nil {
		return nil
	}
	for _, streamSubscription := range api.exchangeRateSubscriptions {
		streamSubscription.cancel()
	}
	api.stoppedWG.Wait()
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
	client      grpc.ServerStreamingClient[R]
	lastReply   *T
	recordReply func(*streamSubscription[R, T], *R)
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
	logger      *slog.Logger
	mutex       sync.RWMutex
}

func (s *streamSubscription[R, T]) IsClosed() bool {
	if s == nil {
		return true
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

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
		reply, err := s.client.Recv()
		if err != nil {
			s.markClosed()
			if s.ctx.Err() != nil || s.client.Context().Err() != nil {
				s.logger.Info("closing subscription; client shutting down")
			} else if errors.Is(err, io.EOF) {
				s.logger.Info("closing subscription; server closed connection")
			} else {
				s.logger.Info("closing subscription; recv failure", slog.Any("err", err))
			}
			return
		}
		s.recordReply(s, reply)
	}
}

func (s *streamSubscription[R, T]) LastReply() (*T, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastReply == nil {
		return nil, finance.ErrRequestPending
	}
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

func recordSecurityMarketDataReply(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], symbol *finance.Symbol, reply *proto.SecurityMarketDataReply) {
	if reply.Error != nil {
		s.logger.Debug("ignoring errornous market data reply", slog.Any("err", reply.Error))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("recording security market data reply")
	s.lastReply = securityMarketDataReplyToQuote(symbol, reply)
}
