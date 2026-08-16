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
	"strings"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
)

func currencyRateReplyToExchangeRate(reply *proto.CurrencyRateReply) *finance.ExchangeRate {
	if reply == nil {
		return nil
	}
	now := time.Now()
	return &finance.ExchangeRate{
		Timestamp:       now,
		Base:            finance.Currency(reply.CurrencyFrom),
		Quote:           finance.Currency(reply.CurrencyTo),
		Rate:            reply.CurrencyRate,
		Source:          Name,
		SourceTimestamp: now,
	}
}

// securityInfoToSymbol assembles a composite finance.Symbol from a TAPI
// SecurityInfoReply. The reply carries the individual identifiers of a single
// security (ISIN, WKN, domestic and US mnemonics) as separate code entries.
func securityInfoToSymbol(reply *proto.SecurityInfoReply) *finance.Symbol {
	if reply == nil {
		return nil
	}
	symbol := &finance.Symbol{
		Name: reply.GetName(),
		Type: securityClassToSecurityType(reply.GetSecurityClass()),
	}
	var mnemonic, mnemonicUS string
	for _, securityCode := range reply.GetSecurityCodes() {
		switch securityCode.GetCodeType() {
		case proto.SecurityCodeType_ISIN:
			symbol.ISIN = securityCode.GetCode()
		case proto.SecurityCodeType_WKN:
			symbol.WKN = securityCode.GetCode()
		case proto.SecurityCodeType_MNEMONIC:
			mnemonic = securityCode.GetCode()
		case proto.SecurityCodeType_MNEMONIC_US:
			mnemonicUS = securityCode.GetCode()
		}
	}
	if strings.HasPrefix(strings.ToUpper(symbol.ISIN), "US") {
		symbol.Ticker = mnemonicUS
	}
	if symbol.Ticker == "" {
		symbol.Ticker = mnemonic
	}
	return symbol
}

// securityClassToSecurityType maps a TAPI SecurityClass onto a finance.SecurityType.
func securityClassToSecurityType(securityClass proto.SecurityClass) finance.SecurityType {
	switch securityClass {
	case proto.SecurityClass_STOCK:
		return finance.SecurityTypeEquity
	case proto.SecurityClass_TRACKERS:
		return finance.SecurityTypeETF
	default:
		return finance.SecurityTypeUnknown
	}
}

func securityMarketDataReplyToQuote(symbol *finance.Symbol, reply *proto.SecurityMarketDataReply) *finance.Quote {
	if reply == nil {
		return nil
	}
	now := time.Now()
	timestamp := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if reply.LastDateTime != nil {
		timestamp = time.Unix(reply.LastDateTime.Seconds, int64(reply.LastDateTime.Nanos))
	}
	return &finance.Quote{
		Symbol:          *symbol,
		Timestamp:       timestamp,
		Open:            reply.OpenPrice,
		High:            reply.HighPrice,
		Low:             reply.LowPrice,
		Close:           reply.LastPrice,
		Price:           reply.LastPrice,
		Volume:          int64(reply.TodayVolume),
		Currency:        finance.Currency(reply.Currency),
		Source:          Name,
		SourceTimestamp: now,
	}
}
