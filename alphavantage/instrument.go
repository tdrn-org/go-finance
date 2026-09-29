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

const searchMatchScore float64 = 0.5

// See [finance.InstrumentProvider]
func (api *API) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	instruments := make([]finance.Instrument, 0)
	err := api.runSymbolSearch(ctx, query, searchMatchScore, func(bestMatch *bestMatchResponse) error {
		instrument := bestMatch.ToInstrument()
		err := instrument.Validate()
		if err == nil {
			instruments = append(instruments, *instrument)
		} else {
			api.logger.Warn("ignoring invalid instrument", slog.Any("err", err))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return instruments, nil
}

// See [finance.InstrumentProvider]
func (api *API) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	return instruments, nil
}

func (api *API) runSymbolSearch(ctx context.Context, query string, minScore float64, collect func(*bestMatchResponse) error) error {
	response, err := api.getSymbolSearch(ctx, query)
	if err != nil {
		return err
	}
	for _, bestMatch := range response.BestMatches {
		match, err := bestMatch.Match(minScore)
		if err != nil {
			return err
		}
		if !match {
			continue
		}
		err = collect(&bestMatch)
		if err != nil {
			return err
		}
	}
	return nil
}

func (api *API) getSymbolSearch(ctx context.Context, query string) (*symbolSearchResponse, error) {
	apiURL := api.url("function", "SYMBOL_SEARCH", "keywords", query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create symbol search request (cause: %w)", Name, err)
	}
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send symbol search request (cause: %w)", Name, err)
	}
	defer rsp.Body.Close()
	response := &symbolSearchResponse{}
	err = api.decodeAndCheckResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
