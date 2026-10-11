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

// Package consorsbank integrates the [Consorsbank TAPI].
//
// [Consorsbank TAPI]: https://www.consorsbank.de/web/Wertpapierhandel/Trading-Software/ActiveTrader#API
package consorsbank

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Name of the Consorsbank Provider.
const Name string = "consorsbank"

// API provides access to the [Consorsbank TAPI] and implements
//   - [finance.FX]
//   - [finance.ListingProvider]
//
// [Consorsbank TAPI]: https://www.consorsbank.de/web/Wertpapierhandel/Trading-Software/ActiveTrader#API
type API struct {
	address                   string
	tlsConfig                 *tls.Config
	secret                    string
	includeExchanges          []string
	preferredCurrency         finance.Currency
	subscriptionTimeout       time.Duration
	session                   *apiSession
	exchangeRateSubscriptions map[string]*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate]
	quoteSubscriptions        map[string]*streamSubscription[proto.SecurityMarketDataReply, finance.Quote]
	logger                    *slog.Logger
	mutex                     sync.Mutex
	stoppedWG                 sync.WaitGroup
}

// NewAPI creates a new Consorsbank provider instance using the given [Config].
func NewAPI(config Config) (*API, error) {
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
	includeExchanges, err := config.GetIncludeExchanges()
	if err != nil {
		return nil, err
	}
	preferredCurrency, err := config.GetPreferredCurrency()
	if err != nil {
		return nil, err
	}
	subscriptionTimeout, err := config.GetSubscriptionTimeout()
	if err != nil {
		return nil, err
	}
	logger := slog.With(slog.String("provider", Name), slog.String("address", address))
	api := &API{
		address:                   address,
		tlsConfig:                 tlsConfig,
		secret:                    secret,
		includeExchanges:          includeExchanges,
		preferredCurrency:         preferredCurrency,
		subscriptionTimeout:       subscriptionTimeout,
		exchangeRateSubscriptions: make(map[string]*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate]),
		quoteSubscriptions:        make(map[string]*streamSubscription[proto.SecurityMarketDataReply, finance.Quote]),
		logger:                    logger,
	}
	return api, nil
}

// Shutdown requests a Consorsbank provider shutdown by gracefully closing any
// running session.
func (api *API) Shutdown(ctx context.Context) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	return errors.Join(api.shutdownSessionLocked(ctx), api.closeSessionLocked())
}

// Close closes any running session.
func (api *API) Close() error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	return api.closeSessionLocked()
}

// See [finance.APIProvider.ProviderName]
func (api *API) ProviderName() string {
	return Name
}

func (api *API) getSessionLocked(ctx context.Context) (*apiSession, error) {
	if api.session != nil {
		return api.session, nil
	}
	api.logger.Info("creating TAPI session...")
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(api.tlsConfig)),
	}
	grpcClient, err := grpc.NewClient(api.address, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create gRPC client to TAPI at %q (cause: %w)", Name, api.address, err)
	}
	accessService := proto.NewAccessServiceClient(grpcClient)
	loginReply, err := accessService.Login(ctx, &proto.LoginRequest{Secret: api.secret})
	if err != nil {
		grpcClient.Close()
		return nil, fmt.Errorf("%s: failed to login to TAPI at %q (cause: %w)", Name, api.address, err)
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
		return fmt.Errorf("%s: failed to send logout request to TAPI at %q (cause: %w)", Name, api.address, err)
	}
	return nil
}

func (api *API) closeSessionLocked() error {
	if api.session == nil {
		return nil
	}
	api.logger.Info("closing TAPI session...")
	err := api.session.grpcClient.Close()
	api.session = nil
	if err != nil {
		return fmt.Errorf("%s: failed to close gRPC client to TAPI at %q (cause: %w)", Name, api.address, err)
	}
	return nil
}

type apiSession struct {
	grpcClient  *grpc.ClientConn
	AccessToken string
}

func (s *apiSession) SecurityService() proto.SecurityServiceClient {
	return proto.NewSecurityServiceClient(s.grpcClient)
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

	if s.subscriptionTimeout <= 0 {
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
	// Only start extending timeout after at least one reply has has been
	// received, to timeout dysfunctional subscriptions.
	s.suscribeUntil = time.Now().Add(s.subscriptionTimeout)
	return s.lastReply, nil
}
