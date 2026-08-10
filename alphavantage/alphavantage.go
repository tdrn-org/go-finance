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

package alphavantage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/tdrn-org/go-finance"
)

const Name string = "alphavantage"

const defaultBaseURLString string = "https://www.alphavantage.co/query/"

var DefaultBaseURL *url.URL = func() *url.URL {
	defaultBaseURL, err := url.Parse(defaultBaseURLString)
	if err != nil {
		panic(err)
	}
	return defaultBaseURL
}()

type API struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
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
	api := &API{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
		logger:     logger,
	}
	return api, nil
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	apiURL := api.url("function", "CURRENCY_EXCHANGE_RATE", "from_currency", string(base), "to_currency", string(quote))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed create query exchange rate request (cause: %w)", err)
	}
	api.logger.Debug("querying exchange rate", slog.Any("url", req.URL))
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send query exchange rate request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	var response currencyExchangeRateResponse
	err = api.decodeResponse(rsp, &response)
	if err != nil {
		return nil, err
	}
	api.logger.Debug("found exchange rate", slog.String("date", response.RealtimeRate.LastRefreshed), slog.String("base", response.RealtimeRate.FromCurrencyCode), slog.String("quote", response.RealtimeRate.ToCurrencyCode), slog.String("rate", response.RealtimeRate.ExchangeRate))
	return response.ToExchangeRate()
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
