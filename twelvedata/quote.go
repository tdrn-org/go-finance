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

// See [finance.QuoteProvider]
func (api *API) QueryQuote(ctx context.Context, instrument *finance.Instrument) (*finance.Quote, error) {
	ticker, ok := instrument.Identifier(finance.InstrumentIdentifierTicker)
	if !ok {
		return nil, finance.ErrQuoteNotAvailable
	}
	response, rsp, err := api.client.MarketDataAPI.
		GetQuote(ctx).
		Symbol(ticker).
		MicCode(instrument.MIC).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("%s: query quote failure (cause: %w)", Name, err)
	}
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return nil, err
	}
	return getQuoteResponseToQuote(instrument, response)
}
