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

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func TestDemoInstrumentProviderAPI(t *testing.T) {
	api := newDemoAPI(t)

	testInstrumentProviderAPI(t, api)
}

func TestAlphaVantageInstrumentProviderAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testInstrumentProviderAPI(t, api)
}

func TestConsorsbankInstrumentProviderAPI(t *testing.T) {
	api := newConsorsbankAPI(t)

	testInstrumentProviderAPI(t, api)
}

func TestOpenFIGIInstrumentProviderAPI(t *testing.T) {
	api := newOpenFIGIAPI(t)

	testInstrumentProviderAPI(t, api)
}

func TestTwelveDataInstrumentProviderAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testInstrumentProviderAPI(t, api)
}

func testInstrumentProviderAPI(t *testing.T, api finance.InstrumentProvider) {
	t.Log("provider", api.ProviderName())

	// SearchInstruments
	queries := []string{
		"Apple",        // Name
		"AAPL",         // Ticker
		"US0378331005", // ISIN
		"865985",       // WKN
		"BBG000B9Y5X2", // FIGI
	}
	for _, query := range queries {
		searchResult, err := api.SearchInstruments(t.Context(), query)
		if !errors.Is(err, finance.ErrInstrumentSearchRestricted) {
			for _, i := range searchResult {
				fmt.Println(i.String())
			}
			require.NoError(t, err)
			require.NotNil(t, searchResult)
		}
	}

	// ResolveInstruments
	instruments := []finance.Instrument{
		{
			ID: "1",
			Identifiers: map[finance.InstrumentIdentifier]string{
				finance.InstrumentIdentifierTicker: "AAPL",
			},
			MIC: "XNYS",
		},
		{
			ID: "2",
			Identifiers: map[finance.InstrumentIdentifier]string{
				finance.InstrumentIdentifierISIN: "US0378331005",
			},
			MIC: "XNYS",
		},
		{
			ID: "3",
			Identifiers: map[finance.InstrumentIdentifier]string{
				finance.InstrumentIdentifierWKN: "865985",
			},
			MIC: "XNYS",
		},
		{
			ID: "4",
			Identifiers: map[finance.InstrumentIdentifier]string{
				finance.InstrumentIdentifierFIGI: "BBG000B9Y5X2",
			},
			MIC: "XNYS",
		},
	}
	resolveResult, err := api.ResolveInstruments(t.Context(), instruments)
	require.NoError(t, err)
	require.Equal(t, len(instruments), len(resolveResult))
}
