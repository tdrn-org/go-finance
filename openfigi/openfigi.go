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

package openfigi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/tdrn-org/go-finance"
	openfigiapi "github.com/tdrn-org/go-finance/openfigi/api"
)

const Name string = "openfigi"

const defaultBaseURLString string = "https://api.openfigi.com/v3"

var DefaultBaseURL *url.URL = func() *url.URL {
	defaultBaseURL, err := url.Parse(defaultBaseURLString)
	if err != nil {
		panic(err)
	}
	return defaultBaseURL
}()

type API struct {
	baseURL   *url.URL
	apiKey    string
	apiClient openfigiapi.ClientWithResponsesInterface
	logger    *slog.Logger
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
	httpClientOption := func(apiClient *openfigiapi.Client) error {
		apiClient.Client = httpClient
		return nil
	}
	apiClient, err := openfigiapi.NewClientWithResponses(baseURL.String(), httpClientOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client (cause: %w)", err)
	}
	api := &API{
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiClient: apiClient,
		logger:    logger,
	}
	return api, nil
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) SearchSymbol(ctx context.Context, query string) ([]finance.Symbol, error) {
	request := openfigiapi.SearchRequest{
		Query: &query,
	}
	response, err := api.apiClient.PostSearchWithResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send search symbol request (cause: %w)", err)
	}
	err = api.checkHttpStatus(response.HTTPResponse)
	if err != nil {
		return nil, err
	}
	figiResults := *response.JSON200.Data
	symbols := make([]finance.Symbol, 0, len(figiResults))
	for _, figiResult := range figiResults {
		symbol := figiResultToSymbol(&figiResult)
		if symbol == nil {
			continue
		}
		symbols = append(symbols, *symbol)
	}
	return symbols, nil
}

func (api *API) checkHttpStatus(rsp *http.Response) error {
	switch rsp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("service failure (status: %d - %s)", rsp.StatusCode, rsp.Status)
	}
}
