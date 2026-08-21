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

package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance/config"
)

func TestLoadConfig(t *testing.T) {
	_, err := config.Load("testdata/finance.toml", true)
	require.NoError(t, err)
}

func TestNewFXProvider(t *testing.T) {
	config, err := config.Load("testdata/finance.toml", true)
	require.NoError(t, err)
	provider, err := config.NewFXProvider()
	require.NoError(t, err)
	providerName := provider.ProviderName()
	require.Equal(t, "cached:fallback:frankfurter|alphavantage|consorsbank|twelvedata", providerName)
}

func TestNewSymbolsProvider(t *testing.T) {
	config, err := config.Load("testdata/finance.toml", true)
	require.NoError(t, err)
	provider, err := config.NewSymbolsProvider()
	require.NoError(t, err)
	providerName := provider.ProviderName()
	require.Equal(t, "cached:merge:openfigi|alphavantage|consorsbank|twelvedata", providerName)
}

func TestNewEquityProvider(t *testing.T) {
	config, err := config.Load("testdata/finance.toml", true)
	require.NoError(t, err)
	provider, err := config.NewEquityProvider()
	require.NoError(t, err)
	providerName := provider.ProviderName()
	require.Equal(t, "cached:consorsbank", providerName)
}
