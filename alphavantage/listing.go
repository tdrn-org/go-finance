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
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
	"go.opentelemetry.io/otel/attribute"
)

const searchMatchScore float64 = 0.5

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	logger := api.logger.With(slog.String("query", query))
	logger.Info("search listings...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.String("query", query))
	defer span.End()

	listings, err := api.searchListings(traceCtx, query)
	if err != nil {
		logger.Warn("search listings failure", slog.Any("err", err))
	} else {
		logger.Info("listings received", slog.Int("len", len(listings)))
	}
	return listings, tracehelper.SetStatus(span, err)
}

func (api *API) searchListings(ctx context.Context, query string) ([]finance.Listing, error) {
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
	err = api.decodeAndCheckResponse(ctx, rsp, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()))
	logger.Info("querying quote...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.String("listing", listing.ID()))
	defer span.End()

	quote, err := api.queryQuote(traceCtx, listing)
	if err != nil {
		logger.Warn("query quote failure", slog.Any("err", err))
	} else {
		logger.Info("quote received", slog.Float64("price", quote.Price), slog.String("currency", string(quote.Currency)))
	}
	return quote, tracehelper.SetStatus(span, err)
}

func (api *API) queryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
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

func (api *API) getGlobalQuote(ctx context.Context, symbol string) (*quoteResponse, error) {
	apiURL := api.url("function", "GLOBAL_QUOTE", "symbol", symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create global quote request (cause: %w)", Name, err)
	}
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send global quote request (cause: %w)", Name, err)
	}
	defer rsp.Body.Close()
	response := &globalQuoteResponse{}
	err = api.decodeAndCheckResponse(ctx, rsp, response)
	if err != nil {
		return nil, err
	}
	return &response.GlobalQuote, nil
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (api *API) QueryQuoteHistory(ctx context.Context, listing finance.Listing, history finance.History) (finance.Bars, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()), slog.String("history", string(history)))
	logger.Info("querying quote history...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.String("listing", listing.ID()), attribute.String("history", string(history)))
	defer span.End()

	bars, err := api.queryQuoteHistory(traceCtx, listing, history)
	if err != nil {
		logger.Warn("query quote history failure", slog.Any("err", err))
	} else {
		logger.Info("quote history received", slog.Int("lene", len(bars)))
	}
	return bars, tracehelper.SetStatus(span, err)
}

func (api *API) queryQuoteHistory(ctx context.Context, listing finance.Listing, history finance.History) (finance.Bars, error) {
	symbol, err := listing.MustAttribute(finance.ListingAttributeKeySymbol)
	if err != nil {
		return nil, err
	}
	limit, err := api.historyParams(history)
	if err != nil {
		return nil, err
	}
	timeSeriesDaily, err := api.getTimeSeriesDaily(ctx, symbol)
	if err != nil {
		return nil, err
	}
	allBars, err := timeSeriesDaily.ToBars(listing)
	if err != nil {
		return nil, err
	}
	bars := make(finance.Bars, 0, limit)
	allBars.Reversed()(func(i int, bar finance.Bar) bool {
		bars = append(bars, bar)
		return len(bars) < limit
	})
	return bars, nil
}

func (api *API) historyParams(history finance.History) (int, error) {
	switch history {
	case finance.History24h:
		return 1, nil
	case finance.History7D:
		return 7, nil
	case finance.History30D:
		return 30, nil
	default:
		return 0, fmt.Errorf("%s: unrecognized history type: %q", Name, history)
	}
}

func (api *API) getTimeSeriesDaily(ctx context.Context, symbol string) (timeSeriesDaily, error) {
	apiURL := api.url("function", "TIME_SERIES_DAILY", "symbol", symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed create time series daily request (cause: %w)", Name, err)
	}
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to send time series daily request (cause: %w)", Name, err)
	}
	defer rsp.Body.Close()
	response := &timeSeriesDailyResponse{}
	err = api.decodeAndCheckResponse(ctx, rsp, response)
	if err != nil {
		return nil, err
	}
	return response.TimeSeries, nil
}
