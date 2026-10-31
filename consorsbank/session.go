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
	"fmt"

	"github.com/tdrn-org/go-finance/consorsbank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type apiSession struct {
	grpcClient      *grpc.ClientConn
	AccessToken     string
	SecurityService proto.SecurityServiceClient
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
		return nil, fmt.Errorf("%s: failed to create gRPC client (cause: %w)", Name, err)
	}
	accessService := proto.NewAccessServiceClient(grpcClient)
	loginReply, err := accessService.Login(ctx, &proto.LoginRequest{Secret: api.secret})
	if err != nil {
		grpcClient.Close()
		return nil, fmt.Errorf("%s: failed to login (cause: %w)", Name, err)
	}
	api.session = &apiSession{
		grpcClient:      grpcClient,
		AccessToken:     loginReply.GetAccessToken(),
		SecurityService: proto.NewSecurityServiceClient(grpcClient),
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
		return fmt.Errorf("%s: failed to logout (cause: %w)", Name, err)
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
		return fmt.Errorf("%s: failed to close gRPC client (cause: %w)", Name, err)
	}
	return nil
}
