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

package twelvedata

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/tdrn-org/go-finance"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

const Name string = "twelvedata"

const apiVersion string = "last"

type API struct {
	client *twelvedata.APIClient
	logger *slog.Logger
}

func NewAPI(config Config) (*API, error) {
	logger := slog.With(slog.String("provider", Name))
	apiKey, err := config.GetAPIKey()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.GetHttpClient()
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
	api := &API{
		client: client,
		logger: logger,
	}
	return api, nil
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	response, rsp, err := api.client.CurrenciesAPI.
		GetExchangeRate(ctx).
		Symbol(fmt.Sprintf("%s/%s", base, quote)).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("query exchange rate failure (cause: %w)", err)
	}
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	return exchangeRateResponseToExchangeRate(response)
}

var instrumentTypeMap map[string]string = map[string]string{
	"Common Stock": string(finance.SecurityTypeEquity),
}

func (api *API) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	response, rsp, err := api.client.ReferenceDataAPI.
		GetSymbolSearch(ctx).
		Symbol(query).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("search symbol failure (cause: %w)", err)
	}
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	return symbolSearchResponseToSymbols(response)
}

func (api *API) QueryQuote(ctx context.Context, symbol finance.Symbol) (*finance.Quote, error) {
	if !symbol.HasTicker() {
		return nil, finance.ErrQuoteNotAvailable
	}
	response, rsp, err := api.client.MarketDataAPI.
		GetQuote(ctx).
		Symbol(symbol.Ticker).
		MicCode(symbol.Exchange).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("query quote failure (cause: %w)", err)
	}
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	return quoteResponseToQuote(&symbol, response)
}

func (api *API) checkHttpStatus(rsp *http.Response) error {
	switch rsp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("service failure (status: %d %s)", rsp.StatusCode, rsp.Status)
	}
}
