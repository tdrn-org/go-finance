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
	"net/http"

	"github.com/tdrn-org/go-finance"
)

// See [finance.QuoteProvider]
func (api *API) QueryQuote(ctx context.Context, instrument *finance.Instrument) (*finance.Quote, error) {
	symbol, ok := instrument.Identifier(InstrumentIdentifierAlphaVantage)
	if !ok {
		return nil, finance.ErrQuoteNotAvailable
	}
	quoteResponse, err := api.getGlobalQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return quoteResponse.ToQuote(instrument)
}

func (api *API) getGlobalQuote(ctx context.Context, symbol string) (*quoteResponse, error) {
	apiURL := api.url("function", "GLOBAL_QUOTE", "symbol", symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create query quote request (cause: %w)", Name, err)
	}
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send query quote request (cause: %w)", Name, err)
	}
	defer rsp.Body.Close()
	response := &globalQuoteResponse{}
	err = api.decodeAndCheckResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	return &response.GlobalQuote, nil
}
