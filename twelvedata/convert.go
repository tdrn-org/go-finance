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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

func exchangeRateResponseToExchangeRate(response *twelvedata.GetExchangeRate200Response) (*finance.ExchangeRate, error) {
	if response == nil {
		return nil, nil
	}
	baseAndQuote := strings.Split(response.Symbol, "/")
	if len(baseAndQuote) != 2 {
		return nil, fmt.Errorf("unexepcted exchange rate symbol '%s'", response.Symbol)
	}
	exchangeRate := &finance.ExchangeRate{
		Timestamp:       time.Unix(response.Timestamp, 0).UTC(),
		Base:            finance.Currency(baseAndQuote[0]),
		Quote:           finance.Currency(baseAndQuote[1]),
		Rate:            response.Rate,
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
	return exchangeRate, nil
}

func symbolSearchResponseToSymbols(response *twelvedata.GetSymbolSearch200Response) (finance.Symbols, error) {
	if response == nil {
		return nil, nil
	}
	if len(response.Data) == 0 {
		return nil, finance.ErrSymbolNotAvailable
	}
	symbols := make(finance.Symbols, 0, len(response.Data))
	for _, responseItem := range response.Data {
		symbols = append(symbols, finance.Symbol{
			Ticker:   responseItem.Symbol,
			Exchange: responseItem.MicCode,
			Name:     responseItem.InstrumentName,
			Type:     finance.MapSecurityType(responseItem.InstrumentType, instrumentTypeMap),
		})
	}
	return symbols, nil
}

func quoteResponseToQuote(symbol *finance.Symbol, response *twelvedata.GetQuote200Response) (*finance.Quote, error) {
	if response == nil {
		return nil, nil
	}
	timestamp := time.Unix(response.Timestamp, 0).UTC()
	if response.LastQuoteAt != nil {
		timestamp = time.Unix(*response.LastQuoteAt, 0).UTC()
	}
	open, err := stringToFloat64(response.Open, "open")
	if err != nil {
		return nil, err
	}
	high, err := stringToFloat64(response.High, "high")
	if err != nil {
		return nil, err
	}
	low, err := stringToFloat64(response.Low, "low")
	if err != nil {
		return nil, err
	}
	close, err := stringToFloat64(response.Close, "close")
	if err != nil {
		return nil, err
	}
	var volume int64
	if response.Volume != nil {
		volume, err = stringToInt64(*response.Volume, "volume")
		if err != nil {
			return nil, err
		}
	}
	if response.Currency == nil {
		return nil, fmt.Errorf("unable to determine currency for symbol '%s'", symbol.Ticker)
	}
	quote := &finance.Quote{
		Symbol:          *symbol,
		Timestamp:       timestamp,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           close,
		Price:           close,
		Volume:          volume,
		Currency:        finance.Currency(*response.Currency),
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
	return quote, nil
}

func stringToFloat64(s, name string) (float64, error) {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse %s '%s' (cause: %w)", name, s, err)
	}
	return value, nil
}

func stringToInt64(s, name string) (int64, error) {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse %s '%s' (cause: %w)", name, s, err)
	}
	return value, nil
}
