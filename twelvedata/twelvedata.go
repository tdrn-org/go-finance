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

// Package twelvedata integrates the [Twelve Data API].
//
// [Twelve Data API]: https://twelvedata.com/docs/
package twelvedata

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/tdrn-org/go-finance"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

// Name of the Twelve Data provider.
const Name string = "twelvedata"

const apiVersion string = "last"

// API provides access to the [Twelve Data API] and implements
//   - [finance.FX]
//   - [finance.ListingProvider]
//
// [Twelve Data API]: https://twelvedata.com/docs/
type API struct {
	client                 *twelvedata.APIClient
	includeMICs            []string
	includeInstrumentTypes []string
	logger                 *slog.Logger
}

// NewAPI creates a new Twelve Data provider instance using the given [Config].
func NewAPI(config Config) (*API, error) {
	apiKey, err := config.GetAPIKey()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.GetHttpClient()
	if err != nil {
		return nil, err
	}
	includeMICs, err := config.GetIncludeMICs()
	if err != nil {
		return nil, err
	}
	includeInstrumentTypes, err := config.GetIncludeInstrumentTypes()
	if err != nil {
		return nil, err
	}
	cfg := twelvedata.NewConfiguration()
	cfg.AddDefaultHeader("Authorization", fmt.Sprintf("apikey %s", apiKey))
	cfg.AddDefaultHeader("X-API-Version", apiVersion)
	if httpClient != nil {
		cfg.HTTPClient = httpClient
	}
	client := twelvedata.NewAPIClient(cfg)
	logger := slog.With(slog.String("provider", Name))
	api := &API{
		client:                 client,
		includeMICs:            includeMICs,
		includeInstrumentTypes: includeInstrumentTypes,
		logger:                 logger,
	}
	return api, nil
}

// See [finance.APIProvider.ProviderName]
func (api *API) ProviderName() string {
	return Name
}

func (api *API) checkHttpStatus(rsp *http.Response) error {
	switch rsp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusTooManyRequests:
		return finance.ErrRateLimitReached
	default:
		return fmt.Errorf("%s: service failure (status: %s)", Name, rsp.Status)
	}
}
