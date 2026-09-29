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

// Package openfigi utilizes the [OpenFIGI API]
// to provide InstrumentProvider service.
//
// [OpenFIGI API]: https://www.openfigi.com/api
package openfigi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"

	"github.com/tdrn-org/go-finance"
	openfigiapi "github.com/tdrn-org/go-finance/openfigi/api"
)

// Name of OpenFIGI provider.
const Name string = "openfigi"

const defaultBaseURLString string = "https://api.openfigi.com/v3"

// DefaultBaseURL defines the default base URL for the OpenFIGI API.
var DefaultBaseURL *url.URL = func() *url.URL {
	defaultBaseURL, err := url.Parse(defaultBaseURLString)
	if err != nil {
		panic(err)
	}
	return defaultBaseURL
}()

// API provides access to the [OpenFIGI API].
//
// [OpenFIGI API]: https://www.openfigi.com/api
type API struct {
	baseURL               *url.URL
	apiKey                string
	apiClient             openfigiapi.ClientWithResponsesInterface
	mics                  []string
	includeSecurityTypes  []string
	includeSecurityTypes2 []string
	logger                *slog.Logger
}

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
	mics, err := config.GetMICs()
	if err != nil {
		return nil, err
	}
	includeSecurityTypes, err := config.GetIncludeSecurityTypes()
	if err != nil {
		return nil, err
	}
	includeSecurityTypes2, err := config.GetIncludeSecurityTypes2()
	if err != nil {
		return nil, err
	}
	httpClientOption := func(apiClient *openfigiapi.Client) error {
		apiClient.Client = httpClient
		return nil
	}
	apiClient, err := openfigiapi.NewClientWithResponses(baseURL.String(), httpClientOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client (cause: %w)", err)
	}
	api := &API{
		baseURL:               baseURL,
		apiKey:                apiKey,
		apiClient:             apiClient,
		mics:                  mics,
		includeSecurityTypes:  includeSecurityTypes,
		includeSecurityTypes2: includeSecurityTypes2,
		logger:                logger,
	}
	return api, nil
}

// See [finance.APIProvider]
func (api *API) ProviderName() string {
	return Name
}

func (api *API) authenticateRequest(ctx context.Context, request *http.Request) error {
	if api.apiKey != "" {
		request.Header.Set("X-OPENFIGI-APIKEY", api.apiKey)
	}
	return nil
}

func (api *API) wrapSystemError(operation string, err error) error {
	return fmt.Errorf("%s: %s call failure (cause: %w)", Name, operation, err)
}

func (api *API) checkAPIResponse(operation string, httpResponse *http.Response) error {
	switch httpResponse.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusTooManyRequests:
		return finance.ErrRateLimitReached
	default:
		return fmt.Errorf("%s: %s API failure (status: %s)", Name, operation, httpResponse.Status)
	}
}

func (api *API) checkAPIResponseWithBody(operation string, httpResponse *http.Response, body any) error {
	err := api.checkAPIResponse(operation, httpResponse)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(body)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("%s: %s yields empty or unexpected response", Name, operation)
	}
	return nil
}
