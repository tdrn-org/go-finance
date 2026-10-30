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

package twelvedata

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/tdrn-org/go-finance"
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
	"github.com/twelvedata/twelvedata-go/twelvedata"
	"go.opentelemetry.io/otel/attribute"
)

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
	err := api.executeSearch(ctx, query, func(responseItem *twelvedata.SymbolSearchResponseItem) error {
		if !api.includeSymbolSearchResponseItem(responseItem) {
			return nil
		}
		listing, err := symbolSearchResponseItemToListing(responseItem)
		if err == nil {
			listings = append(listings, *listing)
		} else {
			api.logger.Warn("ignoring invalid instrument", slog.Any("err", err))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listings, nil
}

func (api *API) includeSymbolSearchResponseItem(responseItem *twelvedata.SymbolSearchResponseItem) bool {
	if len(api.includeMICs) > 0 && !slices.Contains(api.includeMICs, responseItem.MicCode) {
		return false
	}
	if len(api.includeInstrumentTypes) > 0 && !slices.Contains(api.includeInstrumentTypes, responseItem.InstrumentType) {
		return false
	}
	return true
}

func (api *API) executeSearch(ctx context.Context, query string, collect func(*twelvedata.SymbolSearchResponseItem) error) error {
	response, rsp, err := api.client.ReferenceDataAPI.
		GetSymbolSearch(ctx).
		Symbol(query).
		Execute()
	if err != nil {
		return fmt.Errorf("%s: search symbol failure (cause: %w)", Name, err)
	}
	err = api.checkHttpStatus(ctx, rsp)
	if err != nil {
		return err
	}
	for _, responseItem := range response.Data {
		err = collect(&responseItem)
		if err != nil {
			return err
		}
	}
	return nil
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()))
	logger.Info("querying quote...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.Stringer("listing", listing))
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
	ticker, err := listing.MustAttribute(finance.ListingAttributeKeyTicker)
	if err != nil {
		return nil, err
	}
	mic, err := listing.MustAttribute(finance.ListingAttributeKeyMIC)
	if err != nil {
		return nil, err
	}
	response, rsp, err := api.client.MarketDataAPI.
		GetQuote(ctx).
		Symbol(ticker).
		MicCode(mic).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("%s: query quote failure (cause: %w)", Name, err)
	}
	err = api.checkHttpStatus(ctx, rsp)
	if err != nil {
		return nil, err
	}
	return getQuoteResponseToQuote(listing, response)
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (api *API) QueryQuoteHistory(ctx context.Context, listing finance.Listing, from, to time.Time) ([]finance.Bar, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()), slog.Time("from", from), slog.Time("to", to))
	logger.Info("querying quote history...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.Stringer("listing", listing))
	defer span.End()

	history, err := api.queryQuoteHistory(traceCtx, listing, from, to)
	if err != nil {
		logger.Warn("query quote history failure", slog.Any("err", err))
	} else {
		logger.Info("quote history received", slog.Int("len", len(history)))
	}
	return history, tracehelper.SetStatus(span, err)
}

func (api *API) queryQuoteHistory(ctx context.Context, listing finance.Listing, from, to time.Time) ([]finance.Bar, error) {
	ticker, err := listing.MustAttribute(finance.ListingAttributeKeyTicker)
	if err != nil {
		return nil, err
	}
	mic, err := listing.MustAttribute(finance.ListingAttributeKeyMIC)
	if err != nil {
		return nil, err
	}
	startDate, endDate, interval, err := api.timeSeriesParams(from, to)
	if err != nil {
		return nil, err
	}
	response, rsp, err := api.client.MarketDataAPI.
		GetTimeSeries(ctx).
		Symbol(ticker).
		MicCode(mic).
		StartDate(startDate).
		EndDate(endDate).
		Interval(interval).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("%s: query quote history failure (cause: %w)", Name, err)
	}
	err = api.checkHttpStatus(ctx, rsp)
	if err != nil {
		return nil, err
	}
	return getTimeSeriesResponseToBars(listing, response)
}

func (api *API) timeSeriesParams(from, to time.Time) (string, string, twelvedata.IntervalEnum, error) {
	interval := twelvedata.IntervalEnum("")
	timeLayout := ""
	switch finance.FromToInterval(from, to) {
	case finance.IntervalHourly:
		interval = twelvedata.INTERVALENUM__1H
		timeLayout = time.DateTime
	case finance.IntervalDaily:
		interval = twelvedata.INTERVALENUM__1DAY
		timeLayout = time.DateOnly
	default:
		return "", "", "", fmt.Errorf("invalid history interval: %t - %t", from, to)
	}
	startDate := from.Format(timeLayout)
	endDate := to.Format(timeLayout)
	return startDate, endDate, interval, nil
}
