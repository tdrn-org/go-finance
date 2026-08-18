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

// Package alphavantage uses the Alpha Vantage API (https://www.alphavantage.co/documentation/)
// to implement providers for FX, SymbolSearch, Equity.
package alphavantage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/tdrn-org/go-finance"
)

// Name defines the Alpha Vantage provider name.
const Name string = "alphavantage"

const defaultBaseURLString string = "https://www.alphavantage.co/query/"

// DefaultBaseURL defines the default API url for the Alpha Vantage REST service.
var DefaultBaseURL *url.URL = func() *url.URL {
	defaultBaseURL, err := url.Parse(defaultBaseURLString)
	if err != nil {
		panic(err)
	}
	return defaultBaseURL
}()

// API represents the Alpha Vantage provider.
type API struct {
	baseURL             *url.URL
	apiKey              string
	httpClient          *http.Client
	symbolCurrencyCache map[string]string
	logger              *slog.Logger
	mutex               sync.RWMutex
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
		baseURL:             baseURL,
		apiKey:              apiKey,
		httpClient:          httpClient,
		symbolCurrencyCache: make(map[string]string),
		logger:              logger,
	}
	return api, nil
}

// See [finance.APIProvider]
func (api *API) ProviderName() string {
	return Name
}

// See [finance.FX]
func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	response, err := api.queryExchangeRate(ctx, base, quote)
	if err != nil {
		return nil, err
	}
	return response.ToExchangeRate()
}

func (api *API) queryExchangeRate(ctx context.Context, base, quote finance.Currency) (*currencyExchangeRateResponse, error) {
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
	response := &currencyExchangeRateResponse{}
	err = api.decodeResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	err = response.Validate()
	if err != nil {
		return nil, err
	}
	api.logger.Debug("found exchange rate", slog.String("date", response.RealtimeRate.LastRefreshed), slog.String("base", response.RealtimeRate.FromCurrencyCode), slog.String("quote", response.RealtimeRate.ToCurrencyCode), slog.String("rate", response.RealtimeRate.ExchangeRate))
	return response, nil
}

const minMatchScore float64 = 0.5

// See [finance.SymbolResolver]
func (api *API) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	response, err := api.searchSymbol(ctx, query)
	if err != nil {
		return nil, err
	}
	symbolCurrencies := make([][2]string, 0, len(response.BestMatches))
	for _, bestMatch := range response.BestMatches {
		symbolCurrencies = append(symbolCurrencies, [2]string{bestMatch.Symbol, bestMatch.Currency})
	}
	api.cacheSymbolCurrencies(symbolCurrencies)
	return response.ToMatchingSymbols(minMatchScore)
}

func (api *API) searchSymbol(ctx context.Context, query string) (*symbolSearchResponse, error) {
	apiURL := api.url("function", "SYMBOL_SEARCH", "keywords", query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed create search symbol request (cause: %w)", err)
	}
	api.logger.Debug("searching symbol", slog.Any("url", req.URL))
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send symbol search request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	response := &symbolSearchResponse{}
	err = api.decodeResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	err = response.Validate()
	if err != nil {
		return nil, err
	}
	return response, nil
}

// See [finance.Equity]
func (api *API) QueryQuote(ctx context.Context, symbol finance.Symbol) (*finance.Quote, error) {
	quoteResponse, err := api.queryQuote(ctx, &symbol)
	if err != nil {
		return nil, err
	}
	cachedSymbolCurrency := api.cachedSymbolCurrency(symbol.Ticker)
	if cachedSymbolCurrency == "" {
		overviewResponse, err := api.getOverview(ctx, symbol.Ticker)
		if err != nil {
			return nil, err
		}
		cachedSymbolCurrency = overviewResponse.Currency
		if cachedSymbolCurrency == "" {
			return nil, fmt.Errorf("unable to determine currency for symbol '%s'", symbol.Ticker)
		}
		api.cacheSymbolCurrencies([][2]string{{symbol.Ticker, cachedSymbolCurrency}})
	}
	return quoteResponse.ToQuote(&symbol, cachedSymbolCurrency)
}

func (api *API) queryQuote(ctx context.Context, symbol *finance.Symbol) (*globalQuoteResponse, error) {
	if !symbol.HasTicker() {
		return nil, finance.ErrQuoteNotAvailable
	}
	apiURL := api.url("function", "GLOBAL_QUOTE", "symbol", symbol.Ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed create query quote request (cause: %w)", err)
	}
	api.logger.Debug("querying quote", slog.Any("url", req.URL))
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send query quote request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	response := &globalQuoteResponse{}
	err = api.decodeResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	err = response.Validate()
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (api *API) getOverview(ctx context.Context, symbol string) (*overviewResponse, error) {
	apiURL := api.url("function", "OVERVIEW", "symbol", symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed create get overview request (cause: %w)", err)
	}
	api.logger.Debug("getting overview", slog.Any("url", req.URL))
	rsp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send get overview request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	response := &overviewResponse{}
	err = api.decodeResponse(rsp, response)
	if err != nil {
		return nil, err
	}
	err = response.Validate()
	if err != nil {
		return nil, err
	}
	return response, nil
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

func (api *API) cachedSymbolCurrency(symbol string) string {
	api.mutex.RLock()
	defer api.mutex.RUnlock()

	return api.symbolCurrencyCache[symbol]
}

func (api *API) cacheSymbolCurrencies(entries [][2]string) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	for _, entry := range entries {
		api.symbolCurrencyCache[entry[0]] = entry[1]
	}
}
