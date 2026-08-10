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
	"fmt"
	"strconv"
	"time"

	"github.com/tdrn-org/go-finance"
)

type statusResponse struct {
	ErrorMessage string `json:"Error Message,omitempty"`
	Note         string `json:"Note,omitempty"`
	Information  string `json:"Information,omitempty"`
}

func (r *statusResponse) Validate() error {
	if r.ErrorMessage != "" {
		return fmt.Errorf("API call failure: '%s'", r.ErrorMessage)
	}
	if r.Note != "" {
		return fmt.Errorf("%w: rate limit (req/minut) reached: '%s'", finance.ErrRateLimitReached, r.Note)
	}
	if r.Information != "" {
		return fmt.Errorf("%w: rate limit (req/day) reached: '%s'", finance.ErrRateLimitReached, r.Information)
	}
	return nil
}

type realtimeRateResponse struct {
	FromCurrencyCode string `json:"1. From_Currency Code"`
	FromCurrencyName string `json:"2. From_Currency Name"`
	ToCurrencyCode   string `json:"3. To_Currency Code"`
	ToCurrencyName   string `json:"4. To_Currency Name"`
	ExchangeRate     string `json:"5. Exchange Rate"`
	LastRefreshed    string `json:"6. Last Refreshed"`
	TimeZone         string `json:"7. Time Zone"`
	BidPrice         string `json:"8. Bid Price"`
	AskPrice         string `json:"9. Ask Price"`
}

type currencyExchangeRateResponse struct {
	statusResponse
	RealtimeRate realtimeRateResponse `json:"Realtime Currency Exchange Rate"`
}

func (r *currencyExchangeRateResponse) ToExchangeRate() (*finance.ExchangeRate, error) {
	date, err := time.Parse(time.DateTime, r.RealtimeRate.LastRefreshed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exchange rate date '%s' (cause: %w)", r.RealtimeRate.LastRefreshed, err)
	}
	rate, err := strconv.ParseFloat(r.RealtimeRate.ExchangeRate, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exchange rate '%s' (cause: %w)", r.RealtimeRate.ExchangeRate, err)
	}
	exchangeRate := &finance.ExchangeRate{
		Date:  date,
		Base:  finance.Currency(r.RealtimeRate.FromCurrencyCode),
		Quote: finance.Currency(r.RealtimeRate.ToCurrencyCode),
		Rate:  rate,
	}
	return exchangeRate, nil
}
