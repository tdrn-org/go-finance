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

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
)

func (api *API) getSecurityInfoLocked(ctx context.Context, session *apiSession, securityCode *proto.SecurityCode) (*proto.SecurityInfoReply, error) {
	reply, err := session.SecurityService.GetSecurityInfo(ctx, &proto.SecurityInfoRequest{
		AccessToken:  session.AccessToken,
		SecurityCode: securityCode,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query security info (cause: %w)", Name, err)
	}
	if tapiError := reply.GetError(); tapiError != nil {
		return nil, fmt.Errorf("%s: query security info error (code: %s, message: %s)",
			Name, tapiError.GetCode(), tapiError.GetMessage())
	}
	return reply, nil
}

func (api *API) getSecurityWithStockExchangeLocked(ctx context.Context, session *apiSession, securityCode *proto.SecurityCode, exchange string) (*proto.SecurityWithStockExchange, error) {
	securityInfoReply, err := api.getSecurityInfoLocked(ctx, session, securityCode)
	if err != nil {
		return nil, err
	}
	for _, exchangeInfo := range securityInfoReply.StockExchangeInfos {
		if exchangeInfo.StockExchange.Id == exchange {
			securityWithStockExchange := &proto.SecurityWithStockExchange{
				SecurityCode:  securityCode,
				StockExchange: exchangeInfo.StockExchange,
			}
			return securityWithStockExchange, nil
		}
	}
	return nil, fmt.Errorf("%s: failed to resolve exchange info for %q", Name, exchange)
}

func (api *API) getSecurityPriceHistory(ctx context.Context, session *apiSession, securityWithStockExchange *proto.SecurityWithStockExchange, history finance.History) (*proto.SecurityPriceHistoryReply, error) {
	days, timeResolution, err := api.historyParams(history)
	if err != nil {
		return nil, err
	}
	reply, err := session.SecurityService.GetSecurityPriceHistory(ctx, &proto.SecurityPriceHistoryRequest{
		AccessToken:               session.AccessToken,
		SecurityWithStockexchange: securityWithStockExchange,
		Days:                      days,
		TimeResolution:            timeResolution,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query security price history (cause: %w)", Name, err)
	}
	if tapiError := reply.GetError(); tapiError != nil {
		return nil, fmt.Errorf("%s: query security price history error (code: %s, message: %s)",
			Name, tapiError.GetCode(), tapiError.GetMessage())
	}
	return reply, nil
}

func (api *API) historyParams(history finance.History) (int32, proto.TimeResolution, error) {
	switch history {
	case finance.History24h:
		return 1, proto.TimeResolution_HOUR, nil
	case finance.History7D:
		return 7, proto.TimeResolution_DAY, nil
	case finance.History30D:
		return 15, proto.TimeResolution_DAY, nil
	default:
		return 0, proto.TimeResolution_NO_RESOLUTION, fmt.Errorf("%s: unrecognized history type: %q", Name, history)
	}
}
