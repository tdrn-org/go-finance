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
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tdrn-org/go-finance"
)

type statusChecker interface {
	CheckStatus() error
}

type statusResponse struct {
	ErrorMessage string `json:"Error Message,omitempty"`
	Note         string `json:"Note,omitempty"`
	Information  string `json:"Information,omitempty"`
}

func (r *statusResponse) CheckStatus() error {
	if r.ErrorMessage != "" {
		return fmt.Errorf("%s: API call failure: %q", Name, r.ErrorMessage)
	}
	if r.Note != "" {
		return fmt.Errorf("%s: %w: %s", Name, finance.ErrRateLimitReached, r.Note)
	}
	if r.Information != "" {
		return fmt.Errorf("%s: %w: %s", Name, finance.ErrRateLimitReached, r.Information)
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
	timestamp, err := stringToTimestamp(r.RealtimeRate.LastRefreshed, "exchange rate data")
	if err != nil {
		return nil, err
	}
	rate, err := stringToFloat64(r.RealtimeRate.ExchangeRate, "exchange rate")
	if err != nil {
		return nil, err
	}
	exchangeRate := &finance.ExchangeRate{
		Timestamp:       timestamp,
		Base:            finance.Currency(r.RealtimeRate.FromCurrencyCode),
		Quote:           finance.Currency(r.RealtimeRate.ToCurrencyCode),
		Rate:            rate,
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
	return exchangeRate, nil
}

type bestMatchResponse struct {
	Symbol      string `json:"1. symbol"`
	Name        string `json:"2. name"`
	Type        string `json:"3. type"`
	Region      string `json:"4. region"`
	MarketOpen  string `json:"5. marketOpen"`
	MarketClose string `json:"6. marketClose"`
	Timezone    string `json:"7. timezone"`
	Currency    string `json:"8. currency"`
	MatchScore  string `json:"9. matchScore"`
}

func (r *bestMatchResponse) Match(minScore float64) (bool, error) {
	matchScore, err := stringToFloat64(r.MatchScore, "match score")
	if err != nil {
		return false, err
	}
	return matchScore >= minScore, nil
}

//go:embed typeMap.json
var typeMapData []byte

var typeMap map[string]finance.InstrumentType = func() map[string]finance.InstrumentType {
	var typeMap map[string]finance.InstrumentType
	err := json.Unmarshal(typeMapData, &typeMap)
	if err != nil {
		panic(err)
	}
	return typeMap
}()

func (r *bestMatchResponse) ToListing() (*finance.Listing, error) {
	currency := finance.Currency(r.Currency)
	listing, err := finance.NewListing(Name, finance.ListingAttributeKeySymbol).
		InstrumentName(r.Name).
		InstrumentType(typeMap[r.Type]).
		Attribute(finance.ListingAttributeKeySymbol, r.Symbol).
		Currency(&currency).
		Build().Validate()
	if err != nil {
		return nil, err
	}
	return listing, nil
}

type symbolSearchResponse struct {
	statusResponse
	BestMatches []bestMatchResponse `json:"bestMatches"`
}

type quoteResponse struct {
	Symbol           string `json:"01. symbol"`
	Open             string `json:"02. open"`
	High             string `json:"03. high"`
	Low              string `json:"04. low"`
	Price            string `json:"05. price"`
	Volume           string `json:"06. volume"`
	LatestTradingDay string `json:"07. latest trading day"`
	PreviousClose    string `json:"08. previous close"`
	Change           string `json:"09. change"`
	ChangePercent    string `json:"10. change percent"`
}

func (r *quoteResponse) ToQuote(listing finance.Listing) (*finance.Quote, error) {
	currency := listing.Currency()
	if currency == nil {
		return nil, fmt.Errorf("%s: no currency for listing %q", Name, listing.String())
	}
	timestamp, err := stringToTimestamp(r.LatestTradingDay, "latest trading day")
	if err != nil {
		return nil, err
	}
	open, err := stringToFloat64(r.Open, "open")
	if err != nil {
		return nil, err
	}
	high, err := stringToFloat64(r.High, "high")
	if err != nil {
		return nil, err
	}
	low, err := stringToFloat64(r.Low, "low")
	if err != nil {
		return nil, err
	}
	price, err := stringToFloat64(r.Price, "price")
	if err != nil {
		return nil, err
	}
	previousClose, err := stringToFloat64(r.PreviousClose, "previous close")
	if err != nil {
		return nil, err
	}
	volume, err := stringToInt64(r.Volume, "volume")
	if err != nil {
		return nil, err
	}
	quote := &finance.Quote{
		Listing:         listing,
		Timestamp:       timestamp,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           previousClose,
		Price:           price,
		Volume:          volume,
		Currency:        *currency,
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
	return quote, nil
}

type globalQuoteResponse struct {
	statusResponse
	GlobalQuote quoteResponse `json:"Global Quote"`
}
