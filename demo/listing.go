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

package demo

import (
	"context"
	"time"

	"github.com/tdrn-org/go-finance"
)

var demoProviderListing finance.ProviderListing = finance.NewProviderListing(Name).
	InstrumentName("Apple Inc.").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyTicker, "AAPL").
	Attribute(finance.ListingAttributeKeyMIC, "XNGS").
	Build()

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.ProviderListing, error) {
	if !api.enabled {
		return nil, finance.ErrListingSearchRestricted
	}
	listings := []finance.ProviderListing{demoProviderListing}
	return listings, nil
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.ProviderListing) (*finance.Quote, error) {
	if !api.enabled {
		return nil, finance.ErrQuoteNotAvailable
	}
	now := time.Now()
	quote := &finance.Quote{
		Listing:         listing,
		Timestamp:       now,
		Open:            101.0,
		High:            102.0,
		Low:             100.5,
		Close:           100.0,
		Price:           101.0 + float64(now.Minute())/100.0,
		Volume:          int64(now.Hour()),
		Currency:        finance.CurrencyUSD,
		Source:          Name,
		SourceTimestamp: now,
	}
	return quote, nil
}
