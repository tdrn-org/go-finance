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
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
)

func currencyRateReplyToExchangeRate(reply *proto.CurrencyRateReply) *finance.ExchangeRate {
	if reply == nil {
		return nil
	}
	now := time.Now().UTC()
	return &finance.ExchangeRate{
		Timestamp:       now,
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

func securityInfoToProviderListing(reply *proto.SecurityInfoReply, stockExchangeInfo *proto.SecurityStockExchangeInfo) (*finance.ProviderListing, error) {
	if reply == nil {
		return nil, nil
	}
	newListing := finance.NewProviderListing(Name).
		InstrumentName(reply.Name).
		InstrumentType(securityClassMap[reply.SecurityClass]).
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

func securityMarketDataReplyToQuote(listing finance.ProviderListing, reply *proto.SecurityMarketDataReply) *finance.Quote {
	if reply == nil {
		return nil
	}
	now := time.Now().UTC()
	timestamp := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if reply.LastDateTime != nil {
		timestamp = time.Unix(reply.LastDateTime.Seconds, int64(reply.LastDateTime.Nanos)).UTC()
	}
	return &finance.Quote{
		Listing:         listing,
		Timestamp:       timestamp,
		Open:            reply.OpenPrice,
		High:            reply.HighPrice,
		Low:             reply.LowPrice,
		Close:           reply.PreviousPrice,
		Price:           reply.LastPrice,
		Volume:          int64(reply.TodayVolume),
		Currency:        finance.Currency(reply.Currency),
		Source:          Name,
		SourceTimestamp: now,
	}
}
