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
	"github.com/tdrn-org/go-finance/alphavantage"
)

func TestDemoQuoteProviderAPI(t *testing.T) {
	api := newDemoAPI(t)

	testQuoteProviderAPI(t, api)
}

func TestAlphaVantageQuoteProviderAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testQuoteProviderAPI(t, api)
}

func TestConsorsbankQuoteProviderAPI(t *testing.T) {
	api := newConsorsbankAPI(t)

	testQuoteProviderAPI(t, api)
}

func TestTwelveDataQuoteProviderAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testQuoteProviderAPI(t, api)
}

func testQuoteProviderAPI(t *testing.T, api finance.QuoteProvider) {
	t.Log("provider", api.ProviderName())
	currency := finance.CurrencyUSD
	instrument := &finance.Instrument{
		ID: finance.NewInstrumentID(),
		Identifiers: map[finance.InstrumentIdentifier]string{
			finance.InstrumentIdentifierTicker:            "AAPL",
			finance.InstrumentIdentifierISIN:              "US0378331005",
			finance.InstrumentIdentifierFIGI:              "BBG000B9Y5X2",
			alphavantage.InstrumentIdentifierAlphaVantage: "AAPL",
		},
		MIC:      "XNGS",
		Currency: &currency,
	}
	retries := 3
	retrySleep := 500 * time.Millisecond
	for {
		quote, err := api.QueryQuote(t.Context(), instrument)
		if errors.Is(err, finance.ErrRequestPending) {
			retries--
			if retries > 0 {
				time.Sleep(retrySleep)
				continue
			}
		}
		require.NoError(t, err)
		require.NotNil(t, quote)
		fmt.Println(quote)
		break
	}
}
