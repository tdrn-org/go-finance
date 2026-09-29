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

package alphavantage

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/tdrn-org/go-finance"
)

// See [finance.FX]
func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	response, err := api.queryExchangeRate(ctx, base, quote)
	if err != nil {
		return nil, err
	}
	return response.ToExchangeRate()
}

func (api *API) queryExchangeRate(ctx context.Context, base, quote finance.Currency) (*currencyExchangeRateResponse, error) {
	apiURL := api.url("function", "CURRENCY_EXCHANGE_RATE", "from_currency", string(base), "to_currency", string(quote))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create query exchange rate request (cause: %w)", Name, err)
	}
	api.logger.Debug("querying exchange rate", slog.Any("url", api.baseURL))
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send query exchange rate request (cause: %w)", Name, err)
	}
	defer rsp.Body.Close()
	response := &currencyExchangeRateResponse{}
	err = api.decodeAndCheckResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	api.logger.Debug("found exchange rate", slog.String("date", response.RealtimeRate.LastRefreshed), slog.String("base", response.RealtimeRate.FromCurrencyCode), slog.String("quote", response.RealtimeRate.ToCurrencyCode), slog.String("rate", response.RealtimeRate.ExchangeRate))
	return response, nil
}
