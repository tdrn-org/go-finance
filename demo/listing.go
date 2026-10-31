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
	"fmt"
	"time"

	"github.com/tdrn-org/go-finance"
)

var demoListing finance.Listing = *finance.NewListing(Name, finance.ListingAttributeKeyTicker, finance.ListingAttributeKeyMIC).
	InstrumentName("Apple Inc.").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyTicker, "AAPL").
	Attribute(finance.ListingAttributeKeyMIC, "XNGS").
	Build().MustValidate()

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	if !api.enabled {
		return nil, finance.ErrListingSearchRestricted
	}
	listings := []finance.Listing{demoListing}
	return listings, nil
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	if !api.enabled {
		return nil, finance.ErrQuoteNotAvailable
	}

	now := time.Now().UTC()
	quote := api.generateQuote(listing, now)
	return quote, nil
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (api *API) QueryQuoteHistory(ctx context.Context, listing finance.Listing, history finance.History) (finance.Bars, error) {
	delta := 0 * time.Hour
	count := 0
	switch history {
	case finance.History24h:
		delta = 1 * time.Hour
		count = 24
	case finance.History7D:
		delta = 24 * time.Hour
		count = 7
	case finance.History30D:
		delta = 24 * time.Hour
		count = 30
	default:
		return nil, fmt.Errorf("%s: unrecognized history type: %q", Name, history)
	}
	bars := make(finance.Bars, 0, count)
	timestamp := time.Now().UTC()
	for range count {
		bars = append(bars, api.generateQuote(listing, timestamp).Bar)
		timestamp = timestamp.Add(delta)
	}
	return bars, nil
}

func (api *API) generateQuote(listing finance.Listing, timestamp time.Time) *finance.Quote {
	return &finance.Quote{
		Listing:         listing,
		Timestamp:       timestamp,
		Price:           85.0 + float64(timestamp.Day())/100.0,
		Open:            100.0,
		High:            120.0,
		Low:             80.0,
		Volume:          int64(timestamp.Day() * 1000),
		PreviousClose:   100.0,
		Currency:        finance.CurrencyUSD,
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
}
