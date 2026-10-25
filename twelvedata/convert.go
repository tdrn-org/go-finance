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
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

func exchangeRateResponseToExchangeRate(response *twelvedata.GetExchangeRate200Response) (*finance.ExchangeRate, error) {
	baseAndQuote := strings.Split(response.Symbol, "/")
	if len(baseAndQuote) != 2 {
		return nil, fmt.Errorf("%s: unexepcted exchange rate symbol %q", Name, response.Symbol)
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

//go:embed instrumentTypeMap.json
var instrumentTypeMapData []byte

var instrumentTypeMap map[string]finance.InstrumentType = func() map[string]finance.InstrumentType {
	var instrumentTypeMap map[string]finance.InstrumentType
	err := json.Unmarshal(instrumentTypeMapData, &instrumentTypeMap)
	if err != nil {
		panic(err)
	}
	return instrumentTypeMap
}()

func instrumentTypeToInstrumentType(instrumentType string) finance.InstrumentType {
	mapped, ok := instrumentTypeMap[instrumentType]
	if !ok {
		defaultLogger.Warn("missing InstrumentType mapping", slog.String("instrumentType", instrumentType))
		return finance.InstrumentTypeUnknown
	}
	return mapped
}

func symbolSearchResponseItemToListing(responseItem *twelvedata.SymbolSearchResponseItem) (*finance.Listing, error) {
	currency := finance.Currency(responseItem.Currency)
	listing, err := finance.NewListing(Name, idKeys...).
		InstrumentName(responseItem.InstrumentName).
		InstrumentType(instrumentTypeToInstrumentType(responseItem.InstrumentType)).
		Attribute(finance.ListingAttributeKeyTicker, responseItem.Symbol).
		Attribute(finance.ListingAttributeKeyMIC, responseItem.MicCode).
		Currency(&currency).
		Build().Validate()
	if err != nil {
		return nil, err
	}
	return listing, nil
}

func getQuoteResponseToQuote(listing finance.Listing, response *twelvedata.GetQuote200Response) (*finance.Quote, error) {
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
	price, err := stringToFloat64(response.Close, "close")
	if err != nil {
		return nil, err
	}
	previousClose, err := stringToFloat64(response.PreviousClose, "previous close")
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
		return nil, fmt.Errorf("%s: unable to determine quote currency for listing %s", Name, listing.ID())
	}
	currency := finance.Currency(*response.Currency)
	quote := &finance.Quote{
		Listing:         listing,
		Timestamp:       timestamp,
		Open:            open,
		High:            high,
		Low:             low,
		PreviousClose:   previousClose,
		Price:           price,
		Volume:          volume,
		Currency:        currency,
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
	return quote, nil
}

func stringToFloat64(s, name string) (float64, error) {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, fmt.Errorf("%s: failed to parse %s %q (cause: %w)", Name, name, s, err)
	}
	return value, nil
}

func stringToInt64(s, name string) (int64, error) {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0.0, fmt.Errorf("%s: failed to parse %s %q (cause: %w)", Name, name, s, err)
	}
	return value, nil
}
