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

	"github.com/tdrn-org/go-finance"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.ProviderListing, error) {
	api.logger.Info("search listings...", slog.String("query", query))
	listings := make([]finance.ProviderListing, 0)
	err := api.executeSearch(ctx, query, func(responseItem *twelvedata.SymbolSearchResponseItem) error {
		if !api.includeSymbolSearchResponseItem(responseItem) {
			return nil
		}
		listing, err := symbolSearchResponseItemToProviderListing(responseItem)
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
	err = api.checkHttpStatus(rsp)
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
func (api *API) QueryQuote(ctx context.Context, listing finance.ProviderListing) (*finance.Quote, error) {
	err := listing.CheckProvider(Name)
	if err != nil {
		return nil, err
	}
	api.logger.Info("querying quote...", slog.Any("listing", listing))
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
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	return getQuoteResponseToQuote(listing, response)
}
