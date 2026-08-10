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

package frankfurter

import (
	"fmt"
	"time"

	"github.com/tdrn-org/go-finance"
)

type rateResponse struct {
	Date  string           `json:"date"`
	Base  finance.Currency `json:"base"`
	Quote finance.Currency `json:"quote"`
	Rate  float64          `json:"rate"`
}

func (rate *rateResponse) ToExchangeRate() (*finance.ExchangeRate, error) {
	date, err := time.Parse(time.DateOnly, rate.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exchange rate date '%s' (cause: %w)", rate.Date, err)
	}
	exchangeRate := &finance.ExchangeRate{
		Date:  date,
		Base:  rate.Base,
		Quote: rate.Quote,
		Rate:  rate.Rate,
	}
	return exchangeRate, nil
}

type ratesResponse []rateResponse

func (rates ratesResponse) LookupQuote(quote finance.Currency) *rateResponse {
	for _, rate := range rates {
		if rate.Quote == quote {
			return &rate
		}
	}
	return nil
}
