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
	"time"

	"github.com/tdrn-org/go-finance"
)

// See [finance.FX.QueryExchangeRate]
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
