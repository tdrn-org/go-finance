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

func TestDemoCurrencyAPI(t *testing.T) {
	api := newDemoAPI(t)

	testCurrencyAPI(t, api)
}

func TestAlphaVantageCurrencyAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testCurrencyAPI(t, api)
}

func TestConsorsbankCurrencyAPI(t *testing.T) {
	api := newConsorsbankAPI(t)
	defer func() {
		api.Shutdown(t.Context())
		api.Close()
	}()

	testCurrencyAPI(t, api)
}

func TestFranfurterCurrencyAPI(t *testing.T) {
	api := newFrankfurterAPI(t)

	testCurrencyAPI(t, api)
}

func TestTwelveDataCurrencyAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testCurrencyAPI(t, api)
}

func testCurrencyAPI(t *testing.T, api finance.FX) {
	t.Log("provider", api.ProviderName())
	retries := 3
	retrySleep := 500 * time.Millisecond
	for {
		exchangeRate, err := api.QueryExchangeRate(t.Context(), finance.CurrencyUSD, finance.CurrencyEUR)
		if errors.Is(err, finance.ErrRequestPending) {
			retries--
			if retries > 0 {
				time.Sleep(retrySleep)
				continue
			}
		}
		require.NoError(t, err)
		require.Equal(t, finance.CurrencyUSD, exchangeRate.Base)
		require.Equal(t, finance.CurrencyEUR, exchangeRate.Quote)
		require.NotZero(t, exchangeRate.Rate)
		return
	}
}
