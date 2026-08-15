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
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const Name string = "consorsbank"

type API struct {
	address   string
	tlsConfig *tls.Config
	secret    string
	session   *apiSession
	logger    *slog.Logger
	mutex     sync.Mutex
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
		address:   address,
		tlsConfig: tlsConfig,
		secret:    secret,
		logger:    logger,
	}
	return api, nil
}

func (api *API) Shutdown(ctx context.Context) error {
	return api.shutdownSession(ctx)
}

func (api *API) Close() error {
	return api.closeSession()
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) SearchSymbol(ctx context.Context, query string) ([]finance.Symbol, error) {
	session, err := api.getSession(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCodes := api.resolveSecurityCodes(query)
	if len(securityCodes) == 0 {
		return nil, finance.ErrSymbolNotAvailable
	}
	symbols := make([]finance.Symbol, 0)
	for _, securityCode := range securityCodes {
		reply, err := securityService.GetSecurityInfo(ctx, &proto.SecurityInfoRequest{
			AccessToken:  session.AccessToken,
			SecurityCode: securityCode,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query security info from Consorsbank TAPI (cause: %w)", err)
		}
		for _, resolvedSecurityCode := range reply.SecurityCodes {
			symbol := &finance.Symbol{
				Name: reply.Name,
			}
			switch resolvedSecurityCode.CodeType {
			case proto.SecurityCodeType_ISIN:
				symbol.ISIN = resolvedSecurityCode.Code
			case proto.SecurityCodeType_WKN:
				symbol.WKN = resolvedSecurityCode.Code
			case proto.SecurityCodeType_MNEMONIC, proto.SecurityCodeType_MNEMONIC_US:
				symbol.Ticker = resolvedSecurityCode.Code
			}
			symbols = append(symbols, *symbol)
		}
		if len(symbols) > 0 {
			break
		}
	}
	return symbols, nil
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

func (api *API) getSession(ctx context.Context) (*apiSession, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

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

func (api *API) shutdownSession(ctx context.Context) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

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

func (api *API) closeSession() error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if api.session == nil {
		return nil
	}
	err := api.session.grpcClient.Close()
	if err != nil {
		return fmt.Errorf("failed to close Consorsbank TAPI gRPC client (cause: %w)", err)
	}
	return nil
}
