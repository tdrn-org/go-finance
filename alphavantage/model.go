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
		return fmt.Errorf("%w: %s", finance.ErrRateLimitReached, r.Note)
	}
	if r.Information != "" {
		return fmt.Errorf("%w: %s", finance.ErrRateLimitReached, r.Information)
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

type symbolSearchResponse struct {
	statusResponse
	BestMatches []bestMatchResponse `json:"bestMatches"`
}

func (r *symbolSearchResponse) ToMatchingSymbols(minScore float64, hint *finance.Symbol) (finance.Symbols, [][2]string, error) {
	symbols := make(finance.Symbols, 0, len(r.BestMatches))
	symbolCurrencies := make([][2]string, 0, len(r.BestMatches))
	for _, bestMatch := range r.BestMatches {
		matchScore, err := stringToFloat64(bestMatch.MatchScore, "match score")
		if err != nil {
			return nil, nil, err
		}
		if matchScore < minScore {
			continue
		}
		symbols = append(symbols, finance.Symbol{
			Exchange: hint.Exchange,
			Ticker:   bestMatch.Symbol,
			ISIN:     hint.ISIN,
			WKN:      hint.WKN,
			FIGI:     hint.FIGI,
			Name:     bestMatch.Name,
			Type:     finance.MapSecurityType(bestMatch.Type, map[string]string{}),
		})
		symbolCurrencies = append(symbolCurrencies, [2]string{bestMatch.Symbol, bestMatch.Symbol})
	}
	if len(symbols) == 0 {
		return nil, nil, finance.ErrSymbolNotAvailable
	}
	return symbols, symbolCurrencies, nil
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

type globalQuoteResponse struct {
	statusResponse
	GlobalQuote quoteResponse `json:"Global Quote"`
}

func (r *globalQuoteResponse) ToQuote(symbol *finance.Symbol, currency string) (*finance.Quote, error) {
	timestamp, err := stringToTimestamp(r.GlobalQuote.LatestTradingDay, "latest trading day")
	if err != nil {
		return nil, err
	}
	open, err := stringToFloat64(r.GlobalQuote.Open, "open")
	if err != nil {
		return nil, err
	}
	high, err := stringToFloat64(r.GlobalQuote.High, "high")
	if err != nil {
		return nil, err
	}
	low, err := stringToFloat64(r.GlobalQuote.Low, "low")
	if err != nil {
		return nil, err
	}
	price, err := stringToFloat64(r.GlobalQuote.Price, "price")
	if err != nil {
		return nil, err
	}
	previousClose, err := stringToFloat64(r.GlobalQuote.PreviousClose, "previous close")
	if err != nil {
		return nil, err
	}
	volume, err := stringToInt64(r.GlobalQuote.Volume, "volume")
	if err != nil {
		return nil, err
	}
	quote := &finance.Quote{
		Symbol:          *symbol,
		Timestamp:       timestamp,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           previousClose,
		Price:           price,
		Volume:          volume,
		Currency:        finance.Currency(currency),
		Source:          Name,
		SourceTimestamp: time.Now().UTC(),
	}
	return quote, nil
}

type overviewResponse struct {
	statusResponse
	Symbol                     string `json:"Symbol"`
	AssetType                  string `json:"AssetType"`
	Name                       string `json:"Name"`
	Description                string `json:"Description"`
	CIK                        string `json:"CIK"`
	Exchange                   string `json:"Exchange"`
	Currency                   string `json:"Currency"`
	Country                    string `json:"Country"`
	Sector                     string `json:"Sector"`
	Industry                   string `json:"Industry"`
	Address                    string `json:"Address"`
	OfficialSite               string `json:"OfficialSite"`
	FiscalYearEnd              string `json:"FiscalYearEnd"`
	LatestQuarter              string `json:"LatestQuarter"`
	MarketCapitalization       string `json:"MarketCapitalization"`
	EBITDA                     string `json:"EBITDA"`
	PERatio                    string `json:"PERatio"`
	PEGRatio                   string `json:"PEGRatio"`
	BookValue                  string `json:"BookValue"`
	DividendPerShare           string `json:"DividendPerShare"`
	DividendYield              string `json:"DividendYield"`
	EPS                        string `json:"EPS"`
	RevenuePerShareTTM         string `json:"RevenuePerShareTTM"`
	ProfitMargin               string `json:"ProfitMargin"`
	OperatingMarginTTM         string `json:"OperatingMarginTTM"`
	ReturnOnAssetsTTM          string `json:"ReturnOnAssetsTTM"`
	ReturnOnEquityTTM          string `json:"ReturnOnEquityTTM"`
	RevenueTTM                 string `json:"RevenueTTM"`
	GrossProfitTTM             string `json:"GrossProfitTTM"`
	DilutedEPSTTM              string `json:"DilutedEPSTTM"`
	QuarterlyEarningsGrowthYOY string `json:"QuarterlyEarningsGrowthYOY"`
	QuarterlyRevenueGrowthYOY  string `json:"QuarterlyRevenueGrowthYOY"`
	AnalystTargetPrice         string `json:"AnalystTargetPrice"`
	AnalystRatingStrongBuy     string `json:"AnalystRatingStrongBuy"`
	AnalystRatingBuy           string `json:"AnalystRatingBuy"`
	AnalystRatingHold          string `json:"AnalystRatingHold"`
	AnalystRatingSell          string `json:"AnalystRatingSell"`
	AnalystRatingStrongSell    string `json:"AnalystRatingStrongSell"`
	TrailingPE                 string `json:"TrailingPE"`
	ForwardPE                  string `json:"ForwardPE"`
	PriceToSalesRatioTTM       string `json:"PriceToSalesRatioTTM"`
	PriceToBookRatio           string `json:"PriceToBookRatio"`
	EVToRevenue                string `json:"EVToRevenue"`
	EVToEBITDA                 string `json:"EVToEBITDA"`
	Beta                       string `json:"Beta"`
	Week52High                 string `json:"52WeekHigh"`
	Week52Low                  string `json:"52WeekLow"`
	Day50MovingAverage         string `json:"50DayMovingAverage"`
	Day200MovingAverage        string `json:"200DayMovingAverage"`
	SharesOutstanding          string `json:"SharesOutstanding"`
	SharesFloat                string `json:"SharesFloat"`
	PercentInsiders            string `json:"PercentInsiders"`
	PercentInstitutions        string `json:"PercentInstitutions"`
	DividendDate               string `json:"DividendDate"`
	ExDividendDate             string `json:"ExDividendDate"`
}
