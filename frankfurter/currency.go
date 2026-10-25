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
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
)

// See [finance.FX.QueryExchangeRate]
func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	logger := api.logger.With(slog.String("base", string(base)), slog.String("quote", string(quote)))
	logger.Info("querying exchange rate...")
	traceCtx, span := api.tracer.StartConsuming(ctx)
	defer span.End()

	rate, err := api.queryExchangeRate(traceCtx, base, quote)
	if err != nil {
		logger.Info("query exchange rate failure", slog.Any("err", err))
	} else {
		logger.Info("exchange rate received", slog.Float64("rate", rate.Rate))
	}
	return rate, tracehelper.SetStatus(span, err)
}

func (api *API) queryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	apiURL := api.url("/rates", "base", string(base))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create query exchange rate request (cause: %w)", Name, err)
	}
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send query exchange rate request (cause: %w)", Name, err)
	}
	defer rsp.Body.Close()
	var response ratesResponse
	err = api.decodeResponse(ctx, rsp, &response)
	if err != nil {
		return nil, err
	}
	rate := response.LookupQuote(quote)
	if rate == nil {
		return nil, finance.ErrExchangeRateNotAvailable
	}
	return rate.ToExchangeRate()
}
