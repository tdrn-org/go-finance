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

package finance_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func TestAlphaVantageListingProviderAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testListingProviderAPI(t, api)
}

func TestConsorsbankListingProviderAPI(t *testing.T) {
	api := newConsorsbankAPI(t)

	testListingProviderAPI(t, api)
}

func TestTwelvedataListingProviderAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testListingProviderAPI(t, api)
}

func testListingProviderAPI(t *testing.T, api finance.ListingProvider) {
	t.Log("provider", api.ProviderName())

	// SearchListings
	queries := []string{
		// "Apple", // Name
		// "AAPL", // Ticker
		// "US0378331005", // ISIN
		"865985", // WKN
	}
	searchResults := make([]finance.ProviderListing, 0)
	for _, query := range queries {
		searchResult, err := api.SearchListings(t.Context(), query)
		if !errors.Is(err, finance.ErrListingSearchRestricted) {
			require.NoError(t, err)
			require.NotNil(t, searchResult)
			for _, listing := range searchResult {
				fmt.Println(listing.String())
			}
			searchResults = append(searchResults, searchResult...)
		}
	}

	// QueryQuote
	for _, listing := range searchResults {
		retries := 3
		retrySleep := 500 * time.Millisecond
		for {
			quote, err := api.QueryQuote(t.Context(), listing)
			if errors.Is(err, finance.ErrRequestPending) {
				retries--
				if retries > 0 {
					time.Sleep(retrySleep)
					continue
				}
			}
			require.NoError(t, err)
			require.NotNil(t, quote)
			fmt.Println(quote.String())
			break
		}
	}
}
