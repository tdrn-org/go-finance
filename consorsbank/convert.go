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
		// } else if finance.IsTicker(query) {
		// 	securityCodes = append(securityCodes, &proto.SecurityCode{
		// 		Code:     query,
		// 		CodeType: proto.SecurityCodeType_MNEMONIC_US,
		// 	})
		// 	securityCodes = append(securityCodes, &proto.SecurityCode{
		// 		Code:     query,
		// 		CodeType: proto.SecurityCodeType_MNEMONIC,
		// 	})
	}
	return securityCodes
}

func instrumentToSecurityCode(instrument *finance.Instrument) *proto.SecurityCode {
	if instrument.HasIdentifier(finance.InstrumentIdentifierISIN) {
		isin, _ := instrument.Identifier(finance.InstrumentIdentifierISIN)
		return &proto.SecurityCode{
			Code:     isin,
			CodeType: proto.SecurityCodeType_ISIN,
		}
	}
	if instrument.HasIdentifier(finance.InstrumentIdentifierWKN) {
		wkn, _ := instrument.Identifier(finance.InstrumentIdentifierISIN)
		return &proto.SecurityCode{
			Code:     wkn,
			CodeType: proto.SecurityCodeType_WKN,
		}
	}
	return nil
}

func securityInfoToInstrument(reply *proto.SecurityInfoReply, mic string) *finance.Instrument {
	if reply == nil {
		return nil
	}
	instrument := finance.NewInstrument()
	for _, securityCode := range reply.GetSecurityCodes() {
		switch securityCode.GetCodeType() {
		case proto.SecurityCodeType_ISIN:
			instrument.Identifiers[finance.InstrumentIdentifierISIN] = securityCode.GetCode()
		case proto.SecurityCodeType_WKN:
			instrument.Identifiers[finance.InstrumentIdentifierWKN] = securityCode.GetCode()
		case proto.SecurityCodeType_MNEMONIC, proto.SecurityCodeType_MNEMONIC_US:
			instrument.Identifiers[finance.InstrumentIdentifierTicker] = securityCode.GetCode()
		}
	}
	instrument.Name = reply.Name
	instrument.MIC = mic
	return instrument
}

func securityMarketDataReplyToQuote(instrument *finance.Instrument, reply *proto.SecurityMarketDataReply) *finance.Quote {
	if reply == nil {
		return nil
	}
	now := time.Now().UTC()
	timestamp := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if reply.LastDateTime != nil {
		timestamp = time.Unix(reply.LastDateTime.Seconds, int64(reply.LastDateTime.Nanos)).UTC()
	}
	return &finance.Quote{
		Instrument:      instrument.Clone(),
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
