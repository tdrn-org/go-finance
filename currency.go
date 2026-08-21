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

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNoExchangeRate indicates an exchange rate is not available.
	ErrNoExchangeRate error = errors.New("no exchange rate")
)

// Currency type.
type Currency string

const (
	CurrencyAED Currency = "AED"
	CurrencyAFN Currency = "AFN"
	CurrencyALL Currency = "ALL"
	CurrencyAMD Currency = "AMD"
	CurrencyANG Currency = "ANG"
	CurrencyAOA Currency = "AOA"
	CurrencyARS Currency = "ARS"
	CurrencyAUD Currency = "AUD"
	CurrencyAWG Currency = "AWG"
	CurrencyAZN Currency = "AZN"

	CurrencyBAM Currency = "BAM"
	CurrencyBBD Currency = "BBD"
	CurrencyBDT Currency = "BDT"
	CurrencyBHD Currency = "BHD"
	CurrencyBIF Currency = "BIF"
	CurrencyBMD Currency = "BMD"
	CurrencyBND Currency = "BND"
	CurrencyBOB Currency = "BOB"
	CurrencyBRL Currency = "BRL"
	CurrencyBSD Currency = "BSD"
	CurrencyBTN Currency = "BTN"
	CurrencyBWP Currency = "BWP"
	CurrencyBYN Currency = "BYN"
	CurrencyBZD Currency = "BZD"

	CurrencyCAD Currency = "CAD"
	CurrencyCDF Currency = "CDF"
	CurrencyCHF Currency = "CHF"
	CurrencyCLP Currency = "CLP"
	CurrencyCNH Currency = "CNH"
	CurrencyCNY Currency = "CNY"
	CurrencyCOP Currency = "COP"
	CurrencyCRC Currency = "CRC"
	CurrencyCUP Currency = "CUP"
	CurrencyCVE Currency = "CVE"
	CurrencyCZK Currency = "CZK"

	CurrencyDJF Currency = "DJF"
	CurrencyDKK Currency = "DKK"
	CurrencyDOP Currency = "DOP"
	CurrencyDZD Currency = "DZD"

	CurrencyEGP Currency = "EGP"
	CurrencyERN Currency = "ERN"
	CurrencyETB Currency = "ETB"
	CurrencyEUR Currency = "EUR"

	CurrencyFJD Currency = "FJD"
	CurrencyFKP Currency = "FKP"

	CurrencyGBP Currency = "GBP"
	CurrencyGEL Currency = "GEL"
	CurrencyGGP Currency = "GGP"
	CurrencyGHS Currency = "GHS"
	CurrencyGIP Currency = "GIP"
	CurrencyGMD Currency = "GMD"
	CurrencyGNF Currency = "GNF"
	CurrencyGTQ Currency = "GTQ"
	CurrencyGYD Currency = "GYD"

	CurrencyHKD Currency = "HKD"
	CurrencyHNL Currency = "HNL"
	CurrencyHTG Currency = "HTG"
	CurrencyHUF Currency = "HUF"

	CurrencyIDR Currency = "IDR"
	CurrencyILS Currency = "ILS"
	CurrencyIMP Currency = "IMP"
	CurrencyINR Currency = "INR"
	CurrencyIQD Currency = "IQD"
	CurrencyIRR Currency = "IRR"
	CurrencyISK Currency = "ISK"

	CurrencyJEP Currency = "JEP"
	CurrencyJMD Currency = "JMD"
	CurrencyJOD Currency = "JOD"
	CurrencyJPY Currency = "JPY"

	CurrencyKES Currency = "KES"
	CurrencyKGS Currency = "KGS"
	CurrencyKHR Currency = "KHR"
	CurrencyKMF Currency = "KMF"
	CurrencyKPW Currency = "KPW"
	CurrencyKRW Currency = "KRW"
	CurrencyKWD Currency = "KWD"
	CurrencyKYD Currency = "KYD"
	CurrencyKZT Currency = "KZT"

	CurrencyLAK Currency = "LAK"
	CurrencyLBP Currency = "LBP"
	CurrencyLKR Currency = "LKR"
	CurrencyLRD Currency = "LRD"
	CurrencyLSL Currency = "LSL"
	CurrencyLYD Currency = "LYD"

	CurrencyMAD Currency = "MAD"
	CurrencyMDL Currency = "MDL"
	CurrencyMGA Currency = "MGA"
	CurrencyMKD Currency = "MKD"
	CurrencyMMK Currency = "MMK"
	CurrencyMNT Currency = "MNT"
	CurrencyMOP Currency = "MOP"
	CurrencyMRO Currency = "MRO"
	CurrencyMRU Currency = "MRU"
	CurrencyMUR Currency = "MUR"
	CurrencyMVR Currency = "MVR"
	CurrencyMWK Currency = "MWK"
	CurrencyMXN Currency = "MXN"
	CurrencyMYR Currency = "MYR"
	CurrencyMZN Currency = "MZN"

	CurrencyNAD Currency = "NAD"
	CurrencyNGN Currency = "NGN"
	CurrencyNIO Currency = "NIO"
	CurrencyNOK Currency = "NOK"
	CurrencyNPR Currency = "NPR"
	CurrencyNZD Currency = "NZD"

	CurrencyOMR Currency = "OMR"

	CurrencyPAB Currency = "PAB"
	CurrencyPEN Currency = "PEN"
	CurrencyPGK Currency = "PGK"
	CurrencyPHP Currency = "PHP"
	CurrencyPKR Currency = "PKR"
	CurrencyPLN Currency = "PLN"
	CurrencyPYG Currency = "PYG"

	CurrencyQAR Currency = "QAR"

	CurrencyRON Currency = "RON"
	CurrencyRSD Currency = "RSD"
	CurrencyRUB Currency = "RUB"
	CurrencyRWF Currency = "RWF"

	CurrencySAR Currency = "SAR"
	CurrencySBD Currency = "SBD"
	CurrencySCR Currency = "SCR"
	CurrencySDG Currency = "SDG"
	CurrencySEK Currency = "SEK"
	CurrencySGD Currency = "SGD"
	CurrencySHP Currency = "SHP"
	CurrencySLE Currency = "SLE"
	CurrencySOS Currency = "SOS"
	CurrencySRD Currency = "SRD"
	CurrencySSP Currency = "SSP"
	CurrencySTN Currency = "STN"
	CurrencySVC Currency = "SVC"
	CurrencySYP Currency = "SYP"
	CurrencySZL Currency = "SZL"

	CurrencyTHB Currency = "THB"
	CurrencyTJS Currency = "TJS"
	CurrencyTMT Currency = "TMT"
	CurrencyTND Currency = "TND"
	CurrencyTOP Currency = "TOP"
	CurrencyTRY Currency = "TRY"
	CurrencyTTD Currency = "TTD"
	CurrencyTWD Currency = "TWD"
	CurrencyTZS Currency = "TZS"

	CurrencyUAH Currency = "UAH"
	CurrencyUGX Currency = "UGX"
	CurrencyUSD Currency = "USD"
	CurrencyUYU Currency = "UYU"
	CurrencyUZS Currency = "UZS"

	CurrencyVES Currency = "VES"
	CurrencyVND Currency = "VND"
	CurrencyVUV Currency = "VUV"

	CurrencyWST Currency = "WST"

	CurrencyXAF Currency = "XAF"
	CurrencyXAG Currency = "XAG"
	CurrencyXAU Currency = "XAU"
	CurrencyXCD Currency = "XCD"
	CurrencyXCG Currency = "XCG"
	CurrencyXDR Currency = "XDR"
	CurrencyXOF Currency = "XOF"
	CurrencyXPD Currency = "XPD"
	CurrencyXPF Currency = "XPF"
	CurrencyXPT Currency = "XPT"

	CurrencyYER Currency = "YER"

	CurrencyZAR Currency = "ZAR"
	CurrencyZMW Currency = "ZMW"
	CurrencyZWG Currency = "ZWG"
)

// ExchangeRate defines a currency exchange rate at a current
// point in time and sourced from a specific provider.
type ExchangeRate struct {
	// Timestamp defines the point in time this exchange rate
	// was current according to the provider.
	Timestamp time.Time `json:"timestamp"`
	// Base defines the base currency for this exchange rate.
	Base Currency `json:"base"`
	// Quote defines the quoted currency for this exchange rate.
	Quote Currency `json:"quote"`
	// Rate defines the rate for base to quoted currency.
	Rate float64 `json:"rate"`
	// Sources defines the provider this exchange rate has been
	// queried from.
	Source string `json:"source"`
	// SourceTimestamp defines the time this exchange rate has
	// been queried.
	SourceTimestamp time.Time `json:"source_timestamp"`
}

// FX interface provides functions for querying exchange rates.
type FX interface {
	APIProvider
	// QueryExchangeRate queries the exchange rate for the given base and quote
	// currency.
	QueryExchangeRate(ctx context.Context, base, quote Currency) (*ExchangeRate, error)
}
