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

// Package frankfurter integrates the [Frankfurter REST API] service.
//
// [Frankfurter REST API]: https://frankfurter.dev/
package frankfurter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// Name of Frankfurter Provider.
const Name string = "frankfurter"

type API struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     *slog.Logger
}

func NewAPI(config Config) (*API, error) {
	logger := slog.With(slog.String("provider", Name))
	baseURL, err := config.GetBaseURL()
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
		httpClient: httpClient,
		logger:     logger,
	}
	return api, nil
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) url(service string, args ...string) *url.URL {
	apiURL := api.baseURL.JoinPath(service)
	query := apiURL.Query()
	for i := 0; i < len(args); i += 2 {
		query.Set(args[i], args[i+1])
	}
	apiURL.RawQuery = query.Encode()
	return apiURL
}

func (api *API) checkHttpStatus(rsp *http.Response) error {
	switch rsp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("service failure (status: %d %s)", rsp.StatusCode, rsp.Status)
	}
}

func (api *API) decodeResponse(rsp *http.Response, decoded any) error {
	err := json.NewDecoder(rsp.Body).Decode(decoded)
	if err != nil {
		return fmt.Errorf("failed to decode response body (cause: %w)", err)
	}
	return nil
}
