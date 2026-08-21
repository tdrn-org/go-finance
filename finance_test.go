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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance/alphavantage"
	"github.com/tdrn-org/go-finance/consorsbank"
	"github.com/tdrn-org/go-finance/demo"
	"github.com/tdrn-org/go-finance/frankfurter"
	"github.com/tdrn-org/go-finance/openfigi"
	"github.com/tdrn-org/go-finance/twelvedata"
)

func TestDemoProvider(t *testing.T) {
	api := newDemoAPI(t)

	providerName := api.ProviderName()
	require.Equal(t, "demo", providerName)
}

func newDemoAPI(t *testing.T) *demo.API {
	config := &demo.StaticConfig{
		Enabled: true,
	}
	api, err := demo.NewAPI(config)
	require.NoError(t, err)
	return api
}

func TestAlphaVantageProvider(t *testing.T) {
	api := newAlphaVantageAPI(t)

	providerName := api.ProviderName()
	require.Equal(t, "alphavantage", providerName)
}

func newAlphaVantageAPI(t *testing.T) *alphavantage.API {
	config := &alphavantage.StaticConfig{
		BaseURL: alphavantage.DefaultBaseURL,
		APIKey:  os.Getenv("ALPHAVANTAGE_API_KEY"),
	}
	if config.APIKey == "" {
		t.Skip("No Alpha Vange API key set; skipping tests")
	}
	api, err := alphavantage.NewAPI(config)
	require.NoError(t, err)
	return api
}

func TestConsorsbankProvider(t *testing.T) {
	api := newConsorsbankAPI(t)
	defer func() {
		api.Shutdown(t.Context())
		api.Close()
	}()

	providerName := api.ProviderName()
	require.Equal(t, "consorsbank", providerName)
}

func newConsorsbankAPI(t *testing.T) *consorsbank.API {
	root := os.Getenv("CONSORSBANK_ROOT")
	if root == "" {
		t.Skip("No Consorsbank root cert set; skipping now")
	}
	tlsConfig, err := consorsbank.TLSCAFromFile(root)
	require.NoError(t, err)
	config := &consorsbank.StaticConfig{
		TLSConfig: tlsConfig,
		Secret:    os.Getenv("CONSORSBANK_SECRET"),
	}
	if config.Secret == "" {
		t.Skip("No Consorsbank secret set; skipping now")
	}
	api, err := consorsbank.NewAPI(config)
	require.NoError(t, err)
	return api
}

func TestOpenFIGIProvider(t *testing.T) {
	api := newOpenFIGIAPI(t)

	providerName := api.ProviderName()
	require.Equal(t, "openfigi", providerName)
}

func newOpenFIGIAPI(t *testing.T) *openfigi.API {
	config := &openfigi.StaticConfig{
		BaseURL: openfigi.DefaultBaseURL,
		APIKey:  os.Getenv("OPENFIGI_API_KEY"),
	}
	if config.APIKey == "" {
		t.Skip("No OpenFIGI API key set; skipping tests")
	}
	api, err := openfigi.NewAPI(config)
	require.NoError(t, err)
	return api
}

func TestFrankfurterProvider(t *testing.T) {
	api := newFrankfurterAPI(t)

	providerName := api.ProviderName()
	require.Equal(t, "frankfurter", providerName)
}

func newFrankfurterAPI(t *testing.T) *frankfurter.API {
	config := &frankfurter.StaticConfig{
		BaseURL: frankfurter.DefaultBaseURL,
	}
	api, err := frankfurter.NewAPI(config)
	require.NoError(t, err)
	return api
}

func TestTwelveDataProvider(t *testing.T) {
	api := newTwelveDataAPI(t)

	providerName := api.ProviderName()
	require.Equal(t, "twelvedata", providerName)
}

func newTwelveDataAPI(t *testing.T) *twelvedata.API {
	config := &twelvedata.StaticConfig{
		APIKey: os.Getenv("TWELVEDATA_API_KEY"),
	}
	if config.APIKey == "" {
		t.Skip("No Twelve Data API key set; skipping tests")
	}
	api, err := twelvedata.NewAPI(config)
	require.NoError(t, err)
	return api
}
