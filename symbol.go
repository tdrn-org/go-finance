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
	"regexp"
	"strings"
)

var (
	// ErrSymbolNotAvailable indicates a [Symbol] is not known by a provider.
	ErrSymbolNotAvailable error = errors.New("symbol not available")
	// ErrSymbolSearchRestricted indicates a provider is not able to provide
	// a free-text search, but only for defined identifiers (e.g. ISINs).
	ErrSymbolSearchRestricted error = errors.New("symbol search restricted")
)

// SecurityType classifies a financial instrument.
type SecurityType string

const (
	// SecurityTypeUnknown indicates an unknown financial instrument.
	SecurityTypeUnknown SecurityType = ""
	// SecurityTypeEquity indicates an equity (e.g. Common Stock).
	SecurityTypeEquity SecurityType = "equity"
	// SecurityTypeETF indicates an ETF.
	SecurityTypeETF SecurityType = "etf"
)

// MapSecurityType maps a string to an existing [SecurityType] using the given
// alias map (if avaialable). [SecurityTypeUnknown] is returned in case no
// specific [SecurityType] could be identified.
func MapSecurityType(s string, aliasMap map[string]string) SecurityType {
	mappedS, mapped := aliasMap[s]
	if !mapped {
		mappedS = s
	}
	switch strings.ToLower(mappedS) {
	case string(SecurityTypeEquity):
		return SecurityTypeEquity
	case string(SecurityTypeETF):
		return SecurityTypeETF
	default:
		return SecurityTypeUnknown
	}
}

// Symbol identifies a financial instrument.
// It is a composite of identifiers from different naming systems.
// Not all fields are guaranteed to be populated; a provider fills
// whichever identifiers it supports. Callers should check with
// the Has* methods before relying on a specific field.
type Symbol struct {
	// Exchange gives the MIC-Code of the exchange this Symbols refers to.
	Exchange string `json:"exchange"` // MIC-Code, e.g. "XNAS"
	// Ticker gives the ticker symbol this Symbol refers to.
	Ticker string `json:"ticker"` // e.g. "AAPL"
	// ISIN gives the ISIN id this Symbol refers to.
	ISIN string `json:"isin"` // e.g. "US0378331005"
	// WKN gives the ISIN id this Symbol refers to.
	WKN string `json:"wkn"` // e.g. "865985"
	// FIGI gives the ISIN id this Symbol refers to.
	FIGI string `json:"figi"` // e.g. "BBG000B9Y6W2"
	// Name gives the human-readable name of this financial instrument.
	Name string `json:"name"` // e.g. "Apple Inc."
	// Type gives the type of the financial instrument this Symbol refers to.
	Type SecurityType `json:"type"` // e.g. equity
}

// IsEmpty determines if a [Symbol] has any ids set.
func (s *Symbol) IsEmpty() bool {
	return !s.HasTicker() && !s.HasISIN() && !s.HasWKN() && !s.HasFIGI()
}

// HasExchange indicates whether the [Symbols] Exchange attribute is set.
func (s *Symbol) HasExchange() bool { return s.Exchange != "" }

// HasTicker indicates whether the [Symbols] Ticker attribute is set.
func (s *Symbol) HasTicker() bool { return s.Ticker != "" }

// HasISIN indicates whether the [Symbols] ISIN attribute is set.
func (s *Symbol) HasISIN() bool { return s.ISIN != "" }

// HasWKN indicates whether the [Symbols] WKN attribute is set.
func (s *Symbol) HasWKN() bool { return s.WKN != "" }

// HasFIGI indicates whether the [Symbols] FIGI attribute is set.
func (s *Symbol) HasFIGI() bool { return s.FIGI != "" }

// Symbols defines an array of [Symbol]s.
type Symbols []Symbol

var isinPattern regexp.Regexp = *regexp.MustCompile("^[A-Z]{2}[A-Z0-9]{9}[0-9]$")

type isinValidator struct {
	sum    int
	double bool
}

func (v *isinValidator) Validate(s string) bool {
	v.sum = 0
	v.double = false
	if !isinPattern.MatchString(s) {
		return false
	}
	expected := int(s[11] - '0')
	for i := 10; i >= 0; i-- {
		c := s[i]
		if '0' <= c && c <= '9' {
			cValue := int(c - '0')
			v.shift(cValue)
		} else {
			cValue := int(c - 'A' + 10)
			remainder := cValue % 10
			quotient := cValue / 10
			v.shift(remainder)
			v.shift(quotient)
		}
	}
	actual := (10 - (v.sum % 10)) % 10
	return actual == expected
}

func (v *isinValidator) shift(i int) {
	v.double = !v.double
	summand := i
	if v.double {
		summand *= 2
		if summand > 9 {
			summand = (summand % 10) + 1
		}
	}
	v.sum += summand
}

// IsISIN checks whether the given string represents an ISIN.
// This only verfies whether this a syntactically correct ISIN,
// but not whether this ISIN really exists.
func IsISIN(s string) bool {
	return (&isinValidator{}).Validate(strings.ToUpper(s))
}

// IsWKN checks whether the given string represents a WKN.
// This only verfies whether this a syntactically correct WKN,
// but not whether this WKN really exists.
func IsWKN(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		isAlphanumeric := ('0' <= c && c <= '9') || ('A' <= c && c <= 'Z')
		if !isAlphanumeric || c == 'I' || c == 'O' {
			return false
		}
	}
	return true
}

var figiPattern regexp.Regexp = *regexp.MustCompile("^[BCDFGHJKLMNPQRSTVWXYZ]{2}G[0-9BCDFGHJKLMNPQRSTVWXYZ]{8}[0-9]$")

type figiValidator struct {
	sum    int
	double bool
}

func (v *figiValidator) Validate(s string) bool {
	v.sum = 0
	v.double = false
	if !figiPattern.MatchString(s) {
		return false
	}
	expected := int(s[11] - '0')
	for i := 10; i >= 0; i-- {
		c := s[i]
		if '0' <= c && c <= '9' {
			cValue := int(c - '0')
			v.shift(cValue)
		} else {
			cValue := int(c - 'A' + 10)
			quotient := cValue / 10
			remainder := cValue % 10
			v.shift(quotient)
			v.shift(remainder)
		}
	}
	actual := (10 - (v.sum % 10)) % 10
	return actual == expected
}

func (v *figiValidator) shift(i int) {
	v.double = !v.double
	summand := i
	if v.double {
		summand *= 2
		if summand > 9 {
			summand = (summand / 10) + (summand % 10)
		}
	}
	v.sum += summand
}

// IsFIGI checks whether the given string represents a FIGI.
// This only verfies whether this a syntactically correct FIGI,
// but not whether this FIGI really exists.
func IsFIGI(s string) bool {
	return (&figiValidator{}).Validate(strings.ToUpper(s))
}

// SymbolResolver searches for financial instruments.
type SymbolResolver interface {
	APIProvider

	// SearchSymbol looks up symbols matching the given free-text query (name, ticker, ISIN, WKN, etc.).
	// Providers may restrict search to specific code types (e.g. ISIN/WKN only); such providers return
	// ErrSymbolSearchRestricted when the query contains no code they can resolve.
	SearchSymbol(ctx context.Context, query string) (Symbols, error)
}
