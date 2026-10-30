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

func TestDemoListingProviderAPI(t *testing.T) {
	api := newDemoAPI(t)

	testListingProviderAPI(t, api, "AAPL", "Apple")
}

func TestAlphaVantageListingProviderAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testListingProviderAPI(t, api, "AAPL", "Apple")
}

func TestConsorsbankListingProviderAPI(t *testing.T) {
	api := newConsorsbankAPI(t)

	testListingProviderAPI(t, api, "US0378331005", "865985")
}

func TestTwelvedataListingProviderAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testListingProviderAPI(t, api, "AAPL", "US0378331005", "Apple")
}

func testListingProviderAPI(t *testing.T, api finance.ListingProvider, queries ...string) {
	t.Log("provider", api.ProviderName())

	// SearchListings
	listings := make([]finance.Listing, 0)
	for _, query := range queries {
		searchResult, err := api.SearchListings(t.Context(), query)
		if !errors.Is(err, finance.ErrListingSearchRestricted) {
			require.NoError(t, err)
			require.NotNil(t, searchResult)
			for _, listing := range searchResult {
				fmt.Println(listing.ID())
			}
			listings = append(listings, searchResult...)
		}
	}

	// QueryQuote
	require.NotEmpty(t, listings)
	listing := listings[0]
	err := retryRequestPending(func() error {
		quote, err := api.QueryQuote(t.Context(), listing)
		if err != nil {
			return err
		}
		require.NotNil(t, quote)
		fmt.Println(quote.String())
		return nil
	})
	require.NoError(t, err)

	// QueryQuoteHistory
	to := time.Now()
	from := to.Add(-7 * 24 * time.Hour)
	err = retryRequestPending(func() error {
		history, err := api.QueryQuoteHistory(t.Context(), listing, from, to)
		if err != nil {
			return err
		}
		require.NotNil(t, history)
		for _, quote := range history {
			fmt.Println(quote.String())
		}
		return nil
	})
	require.NoError(t, err)
}
