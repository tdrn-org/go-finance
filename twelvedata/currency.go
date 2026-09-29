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

	"github.com/tdrn-org/go-finance"
)

// See [finance.FX]
func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	response, rsp, err := api.client.CurrenciesAPI.
		GetExchangeRate(ctx).
		Symbol(fmt.Sprintf("%s/%s", base, quote)).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("%s: query exchange rate failure (cause: %w)", Name, err)
	}
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	return exchangeRateResponseToExchangeRate(response)
}
