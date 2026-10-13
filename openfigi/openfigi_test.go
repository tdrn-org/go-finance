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

package openfigi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/openfigi"
)

var testListingISIN finance.ProviderListing = *finance.NewProviderListing(openfigi.Name, finance.ListingAttributeKeyISIN, finance.ListingAttributeKeyMIC).
	InstrumentName("APPLE INC").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyISIN, "US0378331005").
	Attribute(finance.ListingAttributeKeyMIC, "XNGS").
	Build().MustValidate()
var testListingWKN finance.ProviderListing = *finance.NewProviderListing(openfigi.Name, finance.ListingAttributeKeyWKN, finance.ListingAttributeKeyMIC).
	InstrumentName("Siemens AG").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyWKN, "723610").
	Attribute(finance.ListingAttributeKeyMIC, "XFRA").
	Build().MustValidate()
var testListingTicker finance.ProviderListing = *finance.NewProviderListing(openfigi.Name, finance.ListingAttributeKeyTicker, finance.ListingAttributeKeyMIC).
	InstrumentName("INTL BUSINESS MACHINES CORP").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyTicker, "IBM").
	Attribute(finance.ListingAttributeKeyMIC, "XNYS").
	Build().MustValidate()

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
