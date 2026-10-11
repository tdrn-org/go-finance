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

package frankfurter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/tdrn-org/go-finance"
)

func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	apiURL := api.url("/rates", "base", string(base))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed create query exchange rate request (cause: %w)", err)
	}
	api.logger.Debug("querying exchange rate", slog.Any("url", req.URL))
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send query exchange rate request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	var response ratesResponse
	err = api.decodeResponse(rsp, &response)
	if err != nil {
		return nil, err
	}
	rate := response.LookupQuote(quote)
	if rate == nil {
		return nil, finance.ErrExchangeRateNotAvailable
	}
	api.logger.Debug("found exchange rate", slog.String("date", rate.Date), slog.String("base", string(rate.Base)), slog.String("quote", string(rate.Quote)), slog.Float64("rate", rate.Rate))
	return rate.ToExchangeRate()
}
