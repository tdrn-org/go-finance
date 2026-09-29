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

// Package alphavantage utilizes the [Alpha Vantage API]
// to provide FX, SymbolSearch and Equity service.
//
// [Alpha Vantage API]: https://www.alphavantage.co/documentation/
package alphavantage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/tdrn-org/go-finance"
)

// Name defines the Alpha Vantage provider name.
const Name string = "alphavantage"

const InstrumentIdentifierAlphaVantage finance.InstrumentIdentifier = "alphavantage:symbol"

// API provides access to the [Alpha Vantage API] and implements
//   - [finance.FX]
//   - [[]finance.Instrumentearcher]
//   - [finance.Equity]
//
// [Alpha Vantage API]: https://www.alphavantage.co/documentation/
type API struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAPI creates a new Alpha Vantage provider instance using the given [Config].
func NewAPI(config Config) (*API, error) {
	logger := slog.With(slog.String("provider", Name))
	baseURL, err := config.GetBaseURL()
	if err != nil {
		return nil, err
	}
	apiKey, err := config.GetAPIKey()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.GetHttpClient()
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	api := &API{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
		logger:     logger,
	}
	return api, nil
}

// See [finance.APIProvider]
func (api *API) ProviderName() string {
	return Name
}

func (api *API) url(args ...string) *url.URL {
	apiURL := *api.baseURL
	query := apiURL.Query()
	for i := 0; i < len(args); i += 2 {
		query.Set(args[i], args[i+1])
	}
	query.Set("apikey", api.apiKey)
	apiURL.RawQuery = query.Encode()
	return &apiURL
}

func (api *API) decodeAndCheckResponse(rsp *http.Response, decoded any) error {
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: service failure (status: %s)", Name, rsp.Status)
	}
	err := json.NewDecoder(rsp.Body).Decode(decoded)
	if err != nil {
		return fmt.Errorf("%s: failed to decode response body (cause: %w)", Name, err)
	}
	status, ok := decoded.(statusChecker)
	if ok {
		err = status.CheckStatus()
		if err != nil {
			return err
		}
	}
	return nil
}
