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
	exchangeRateSubscriptions map[string]*exchangeRateSubscription
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
		exchangeRateSubscriptions: make(map[string]*exchangeRateSubscription),
		logger:                    logger,
	}
	return api, nil
}

func (api *API) Shutdown(ctx context.Context) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	return api.shutdownSessionLocked(ctx)
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
		err := api.startExchangeRateSubscriptionLocked(ctx, base, quote)
		if err != nil {
			return nil, err
		}
		return nil, finance.ErrRequestPending
	}
	return subscription.LastReply()
}

func (api *API) getExchangeRateSubscriptionLocked(base, quote finance.Currency) *exchangeRateSubscription {
	subscriptionKey := fmt.Sprintf("%s/%s", base, quote)
	subscription := api.exchangeRateSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startExchangeRateSubscriptionLocked(ctx context.Context, base, quote finance.Currency) error {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return err
	}
	subscriptionKey := fmt.Sprintf("%s/%s", base, quote)
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	securityService := session.SecurityService()
	client, err := securityService.StreamCurrencyRate(subscriptionCtx, &proto.CurrencyRateRequest{
		AccessToken:  session.AccessToken,
		CurrencyFrom: string(base),
		CurrencyTo:   string(quote),
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return fmt.Errorf("failed to create exchange rate subscription (cause: %w)", err)
	}
	subscription := &exchangeRateSubscription{
		client: client,
		ctx:    subscriptionCtx,
		cancel: subscriptionCancel,
		logger: api.logger.With(slog.String("exchangeRateSubscription", subscriptionKey)),
	}
	api.exchangeRateSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return nil
}

func (api *API) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCodes := api.resolveSecurityCodes(query)
	if len(securityCodes) == 0 {
		return nil, finance.ErrSymbolSearchRestricted
	}
	symbols := make(finance.Symbols, 0, len(securityCodes))
	for _, securityCode := range securityCodes {
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

func (api *API) QueryQuote(ctx context.Context, symbol finance.Symbol) (*finance.Quote, error) {
	return nil, nil
}

func (api *API) resolveSecurityCodes(query string) []*proto.SecurityCode {
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
	for _, exchangeRateSubscription := range api.exchangeRateSubscriptions {
		exchangeRateSubscription.cancel()
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

type exchangeRateSubscription struct {
	client    grpc.ServerStreamingClient[proto.CurrencyRateReply]
	lastReply *finance.ExchangeRate
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	logger    *slog.Logger
	mutex     sync.RWMutex
}

func (s *exchangeRateSubscription) IsClosed() bool {
	if s == nil {
		return true
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.closed
}

func (s *exchangeRateSubscription) markClosed() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.closed = true
}

func (s *exchangeRateSubscription) Run() {
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
		if reply.Error != nil {
			s.logger.Debug("ignoring errornous reply", slog.Any("err", reply.Error))
			continue
		}
		if reply.CurrencyFrom == "" || reply.CurrencyTo == "" || math.IsNaN(reply.CurrencyRate) {
			s.logger.Debug("ignoring empty reply")
			continue
		}
		s.logger.Info("recording reply")
		s.recordReply(reply)
	}
}

func (s *exchangeRateSubscription) LastReply() (*finance.ExchangeRate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastReply == nil {
		return nil, finance.ErrRequestPending
	}
	return s.lastReply, nil
}

func (s *exchangeRateSubscription) recordReply(reply *proto.CurrencyRateReply) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.lastReply = currencyRateReplyToExchangeRate(reply)
}
