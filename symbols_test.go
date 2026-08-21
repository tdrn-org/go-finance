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

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func TestAlphaVantageSymbolAPI(t *testing.T) {
	api := newAlphaVantageAPI(t)

	testSymbolAPI(t, api)
}

func TestConsorsbankSymbolAPI(t *testing.T) {
	api := newConsorsbankAPI(t)
	defer func() {
		api.Shutdown(t.Context())
		api.Close()
	}()

	testSymbolAPI(t, api)
}

func TestOpenFIGISymbolAPI(t *testing.T) {
	api := newOpenFIGIAPI(t)

	testSymbolAPI(t, api)
}

func TestTwelveDataISymbolAPI(t *testing.T) {
	api := newTwelveDataAPI(t)

	testSymbolAPI(t, api)
}

func testSymbolAPI(t *testing.T, api finance.SymbolResolver) {
	t.Log("provider", api.ProviderName())
	symbols, err := api.SearchSymbol(t.Context(), "Apple")
	if !errors.Is(err, finance.ErrSymbolNotAvailable) {
		require.NoError(t, err)
		require.NotEmpty(t, symbols)
	}
	symbols, err = api.SearchSymbol(t.Context(), "AAPL")
	if !errors.Is(err, finance.ErrSymbolNotAvailable) {
		require.NoError(t, err)
		require.NotEmpty(t, symbols)
	}
	symbols, err = api.SearchSymbol(t.Context(), "US0378331005")
	if !errors.Is(err, finance.ErrSymbolNotAvailable) {
		require.NoError(t, err)
		require.NotEmpty(t, symbols)
	}
	symbols, err = api.SearchSymbol(t.Context(), "865985")
	if !errors.Is(err, finance.ErrSymbolNotAvailable) {
		require.NoError(t, err)
		require.NotEmpty(t, symbols)
	}
	symbols, err = api.SearchSymbol(t.Context(), "BBG000B9XRY4")
	if !errors.Is(err, finance.ErrSymbolNotAvailable) {
		require.NoError(t, err)
		require.NotEmpty(t, symbols)
	}
}

func TestIsISIN(t *testing.T) {
	require.True(t, finance.IsISIN("US0378331005"))
	require.False(t, finance.IsISIN("US0378331006"))
}

func TestIsWKN(t *testing.T) {
	require.True(t, finance.IsWKN("865985"))
	require.False(t, finance.IsWKN("86598I"))
}

func TestIsFIGI(t *testing.T) {
	require.True(t, finance.IsFIGI("BBG000B9XRY4"))
	require.False(t, finance.IsFIGI("BBG000B9XRY5"))
}
