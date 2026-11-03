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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/openfigi"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

var testListingSymbol finance.Listing = *finance.NewListing(openfigi.Name, finance.ListingAttributeKeySymbol).
	InstrumentName("APPLE INC").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeySymbol, "AAPL").
	Build().MustValidate()
var testListingISIN finance.Listing = *finance.NewListing(openfigi.Name, finance.ListingAttributeKeyISIN, finance.ListingAttributeKeyMIC).
	InstrumentName("APPLE INC").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyISIN, "US0378331005").
	Attribute(finance.ListingAttributeKeyMIC, "XNGS").
	Build().MustValidate()
var testListingWKN finance.Listing = *finance.NewListing(openfigi.Name, finance.ListingAttributeKeyWKN, finance.ListingAttributeKeyMIC).
	InstrumentName("Siemens AG").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeyWKN, "723610").
	Attribute(finance.ListingAttributeKeyMIC, "XFRA").
	Build().MustValidate()
var testListingTicker finance.Listing = *finance.NewListing(openfigi.Name, finance.ListingAttributeKeyTicker, finance.ListingAttributeKeyMIC).
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
	// Enable HTTP recording
	maskedAPIKey := "<masked>"
	maskAuthorizationHeader := func(i *cassette.Interaction) error {
		i.Request.Headers["X-Openfigi-Apikey"] = []string{maskedAPIKey}
		return nil
	}
	normalizedBodyMatcher := func(r *http.Request, i cassette.Request) bool {
		_, _ = io.ReadAll(r.Body)
		i.Body = ""
		return cassette.DefaultMatcher(r, i)
	}
	vcr := newHTTPRecorder(t, "openfigi", recorder.WithHook(maskAuthorizationHeader, recorder.AfterCaptureHook), recorder.WithMatcher(normalizedBodyMatcher))

	// Setup API key
	apiKey := os.Getenv("OPENFIGI_API_KEY")
	if !vcr.IsRecording() {
		apiKey = maskedAPIKey
	}

	// Create API
	config := &openfigi.StaticConfig{
		BaseURL:    openfigi.DefaultBaseURL,
		APIKey:     apiKey,
		HttpClient: vcr.GetDefaultClient(),
	}
	api, err := openfigi.NewAPI(config)
	require.NoError(t, err)
	return api
}

func newHTTPRecorder(t *testing.T, provider string, opts ...recorder.Option) *recorder.Recorder {
	vcr, err := recorder.New(filepath.Join("testdata", provider, strings.ReplaceAll(t.Name(), "/", "_")), opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, vcr.Stop())
	})
	return vcr
}
