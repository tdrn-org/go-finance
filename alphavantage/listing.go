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

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	api.logger.Info("search listings...", slog.String("query", query))
	listings := make([]finance.Listing, 0)
	err := api.runSymbolSearch(ctx, query, searchMatchScore, func(bestMatch *bestMatchResponse) error {
		listing, err := bestMatch.ToListing()
		if err == nil {
			listings = append(listings, *listing)
		} else {
			api.logger.Warn("ignoring invalid listing", slog.Any("err", err))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listings, nil
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	err := listing.CheckProvider(Name)
	if err != nil {
		return nil, err
	}
	api.logger.Info("querying quote...", slog.Any("listing", listing))
	err = listing.CheckProvider(Name)
	if err != nil {
		return nil, err
	}
	symbol, err := listing.MustAttribute(finance.ListingAttributeKeySymbol)
	if err != nil {
		return nil, err
	}
	quoteResponse, err := api.getGlobalQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return quoteResponse.ToQuote(listing)
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
