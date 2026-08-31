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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func TestDemoEquityAPI(t *testing.T) {
	api := newDemoAPI(t)

	testEquityAP(t, api)
}

func TestAlphaVantageEquityAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testEquityAP(t, api)
}

func TestConsorsbankEquityAPI(t *testing.T) {
	api := newConsorsbankAPI(t)
	defer func() {
		api.Shutdown(t.Context())
		api.Close()
	}()

	testEquityAP(t, api)
}

func TestTwelveDataEquityAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testEquityAP(t, api)
}

func testEquityAP(t *testing.T, api finance.Equity) {
	t.Log("provider", api.ProviderName())
	symbol := finance.Symbol{
		Exchange: "XNGS",
		Ticker:   "AAPL",
		ISIN:     "US0378331005",
		WKN:      "865985",
		FIGI:     "BBG000B9XRY4",
	}
	resolvedSymbol, err := api.ResolveSymbol(t.Context(), symbol)
	require.NoError(t, err)
	retries := 3
	retrySleep := 500 * time.Millisecond
	for {
		quote, err := api.QueryQuote(t.Context(), *resolvedSymbol)
		if errors.Is(err, finance.ErrRequestPending) {
			retries--
			if retries > 0 {
				time.Sleep(retrySleep)
				continue
			}
		}
		require.NoError(t, err)
		require.NotNil(t, quote)
		return
	}
}
