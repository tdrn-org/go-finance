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

package finance

// InstrumentType classifies a financial instrument.
type InstrumentType string

const (
	// InstrumentTypeUnknown indicates an unknown instrument type.
	InstrumentTypeUnknown InstrumentType = ""
	// InstrumentTypeEquity indicates an equity instrument
	// (e.g. common stock).
	InstrumentTypeEquity InstrumentType = "equity"
	// InstrumentTypeETF indicates an exchange-traded fund (ETF).
	InstrumentTypeETF InstrumentType = "etf"
	// InstrumentTypeFund indicates an investment fund.
	InstrumentTypeFund InstrumentType = "fund"
	// InstrumentTypeBond indicates a bond.
	InstrumentTypeBond InstrumentType = "bond"
	// InstrumentTypeIndex indicates a market index.
	InstrumentTypeIndex InstrumentType = "index"
	// InstrumentTypeForex indicates a foreign exchange instrument.
	InstrumentTypeForex InstrumentType = "forex"
	// InstrumentTypeCrypto indicates a cryptocurrency instrument.
	InstrumentTypeCrypto InstrumentType = "crypto"
	// InstrumentTypeOption indicates an option contract.
	InstrumentTypeOption InstrumentType = "option"
	// InstrumentTypeFuture indicates a futures contract.
	InstrumentTypeFuture InstrumentType = "future"
	// InstrumentTypeCertificate indicates a certificate.
	InstrumentTypeCertificate InstrumentType = "certificate"
)
