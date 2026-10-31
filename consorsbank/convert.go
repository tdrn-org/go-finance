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

package consorsbank

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
)

func currencyRateReplyToExchangeRate(_ *finance.ExchangeRate, reply *proto.CurrencyRateReply) *finance.ExchangeRate {
	now := time.Now().UTC()
	timestamp := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return &finance.ExchangeRate{
		Timestamp:       timestamp,
		Base:            finance.Currency(reply.CurrencyFrom),
		Quote:           finance.Currency(reply.CurrencyTo),
		Rate:            reply.CurrencyRate,
		Source:          Name,
		SourceTimestamp: now,
	}
}

func queryToSecurityCodes(query string) []*proto.SecurityCode {
	securityCodes := make([]*proto.SecurityCode, 0)
	if finance.IsISIN(query) {
		securityCodes = append(securityCodes, &proto.SecurityCode{
			Code:     query,
			CodeType: proto.SecurityCodeType_ISIN,
		})
	} else if finance.IsWKN(query) {
		securityCodes = append(securityCodes, &proto.SecurityCode{
			Code:     query,
			CodeType: proto.SecurityCodeType_WKN,
		})
	}
	return securityCodes
}

func listingToSecurityCodeAndExchange(listing finance.Listing) (*proto.SecurityCode, string, error) {
	isin, err := listing.MustAttribute(finance.ListingAttributeKeyISIN)
	if err != nil {
		return nil, "", err
	}
	securityCode := &proto.SecurityCode{
		Code:     isin,
		CodeType: proto.SecurityCodeType_ISIN,
	}
	exchange, err := listing.MustAttribute(finance.ListingAttributeKeyExchange)
	if err != nil {
		return nil, "", err
	}
	return securityCode, exchange, nil
}

//go:embed securityClassMap.json
var securityClassMapData []byte

var securityClassMap map[proto.SecurityClass]finance.InstrumentType = func() map[proto.SecurityClass]finance.InstrumentType {
	var securityClassMap map[proto.SecurityClass]finance.InstrumentType
	err := json.Unmarshal(securityClassMapData, &securityClassMap)
	if err != nil {
		panic(err)
	}
	return securityClassMap
}()

func securityClassToInstrumentType(securityClass proto.SecurityClass) finance.InstrumentType {
	mapped, ok := securityClassMap[securityClass]
	if !ok {
		defaultLogger.Warn("missing SecurityClass mapping", slog.Any("securityClass", securityClass))
		return finance.InstrumentTypeUnknown
	}
	return mapped
}

func securityInfoToListing(reply *proto.SecurityInfoReply, stockExchangeInfo *proto.SecurityStockExchangeInfo) (*finance.Listing, error) {
	newListing := finance.NewListing(Name, idKeys...).
		InstrumentName(reply.Name).
		InstrumentType(securityClassToInstrumentType(reply.SecurityClass)).
		Attribute(finance.ListingAttributeKeyExchange, stockExchangeInfo.StockExchange.Id)
	for _, securityCode := range reply.GetSecurityCodes() {
		switch securityCode.GetCodeType() {
		case proto.SecurityCodeType_ISIN:
			newListing.Attribute(finance.ListingAttributeKeyISIN, securityCode.Code)
		case proto.SecurityCodeType_WKN:
			newListing.Attribute(finance.ListingAttributeKeyWKN, securityCode.Code)
		case proto.SecurityCodeType_MNEMONIC, proto.SecurityCodeType_MNEMONIC_US:
			newListing.Attribute(finance.ListingAttributeKeyTicker, securityCode.Code)
		}
	}
	listing, err := newListing.Build().Validate()
	if err != nil {
		return nil, err
	}
	return listing, nil
}

func securityMarketDataReplyToQuote(listing finance.Listing, last *finance.Quote, reply *proto.SecurityMarketDataReply) *finance.Quote {
	now := time.Now().UTC()
	quote := &finance.Quote{
		Listing:         listing,
		Currency:        finance.Currency(reply.Currency),
		Source:          Name,
		SourceTimestamp: now,
	}
	if last != nil {
		quote.Timestamp = last.Timestamp
		quote.Open = last.Open
		quote.High = last.High
		quote.Low = last.Low
		quote.Price = last.Price
		quote.Volume = last.Volume
		quote.PreviousClose = last.PreviousClose
	}
	for _, changedField := range reply.ChangedFields {
		switch changedField {
		case proto.SecurityMarketDataReply_LAST_DATE_TIME:
			quote.Timestamp = time.Unix(reply.LastDateTime.Seconds, int64(reply.LastDateTime.Nanos)).UTC()
		case proto.SecurityMarketDataReply_OPEN_PRICE:
			quote.Open = reply.OpenPrice
		case proto.SecurityMarketDataReply_HIGH_PRICE:
			quote.High = reply.HighPrice
		case proto.SecurityMarketDataReply_LOW_PRICE:
			quote.Low = reply.LowPrice
		case proto.SecurityMarketDataReply_PREVIOUS_PRICE:
			quote.PreviousClose = reply.PreviousPrice
		case proto.SecurityMarketDataReply_LAST_PRICE:
			quote.Price = reply.LastPrice
		case proto.SecurityMarketDataReply_TODAY_VOLUME:
			quote.Volume = int64(reply.TodayVolume)
		}
	}
	return quote
}

func securityPriceHistoryReplyToBars(listing finance.Listing, reply *proto.SecurityPriceHistoryReply) finance.Bars {
	now := time.Now()
	bars := make(finance.Bars, 0, len(reply.PriceEntries))
	for _, entry := range reply.PriceEntries {
		bars = append(bars, finance.Bar{
			Listing:         listing,
			Timestamp:       time.Unix(entry.CloseTime.Seconds, int64(entry.CloseTime.Nanos)).UTC(),
			Price:           entry.ClosePrice,
			Open:            entry.OpenPrice,
			High:            entry.HighPrice,
			Low:             entry.LowPrice,
			Volume:          int64(entry.Volume),
			Currency:        finance.Currency(reply.Currency),
			Source:          Name,
			SourceTimestamp: now,
		})
	}
	return bars
}
