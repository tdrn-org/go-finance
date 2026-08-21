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

package demo

import (
	"context"
	"log/slog"
	"time"

	"github.com/tdrn-org/go-finance"
)

const Name string = "demo"

type Config interface {
	GetEnabled() bool
}

type StaticConfig struct {
	Enabled bool
}

func (c *StaticConfig) GetEnabled() bool {
	return c.Enabled
}

type API struct {
	enabled bool
	logger  *slog.Logger
}

func NewAPI(config Config) (*API, error) {
	logger := slog.With(slog.String("provider", Name))
	api := &API{
		enabled: config.GetEnabled(),
		logger:  logger,
	}
	return api, nil
}

func (api *API) ProviderName() string {
	return Name
}

func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	if !api.enabled {
		return nil, finance.ErrExchangeRateNotAvailable
	}
	now := time.Now()
	exchangeRate := &finance.ExchangeRate{
		Timestamp:       now,
		Base:            base,
		Quote:           quote,
		Rate:            1.0 + float64(now.Minute())/100.0,
		Source:          Name,
		SourceTimestamp: now,
	}
	return exchangeRate, nil
}

func (api *API) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	if !api.enabled {
		return nil, finance.ErrSymbolSearchRestricted
	}
	symbol := finance.Symbol{
		Exchange: "BCDS",
		Ticker:   "SNOL",
		ISIN:     "DE1234567890",
		WKN:      "456789",
		FIGI:     "BBG000000001",
		Name:     "SnakeOil Ltd.",
		Type:     finance.SecurityTypeEquity,
	}
	return finance.Symbols{symbol}, nil
}

func (api *API) QueryQuote(ctx context.Context, symbol finance.Symbol) (*finance.Quote, error) {
	if !api.enabled {
		return nil, finance.ErrQuoteNotAvailable
	}
	now := time.Now()
	quote := &finance.Quote{
		Symbol:          symbol,
		Timestamp:       now,
		Open:            101.0,
		High:            102.0,
		Low:             100.5,
		Close:           100.0,
		Price:           101.0 + float64(now.Minute())/100.0,
		Volume:          int64(now.Hour()),
		Currency:        finance.CurrencyUSD,
		Source:          Name,
		SourceTimestamp: now,
	}
	return quote, nil
}
