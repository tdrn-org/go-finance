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
	ErrSymbolNotAvailable error = errors.New("not available")
)

// SecurityType classifies a financial instrument.
type SecurityType string

const (
	SecurityTypeUnknown SecurityType = ""
	SecurityTypeEquity  SecurityType = "equity"
	SecurityTypeETF     SecurityType = "etf"
)

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
	Exchange string       // MIC-Code, e.g. "XNAS"
	Ticker   string       // e.g. "AAPL"
	ISIN     string       // e.g. "US0378331005"
	WKN      string       // e.g. "865985"
	FIGI     string       // e.g. "BBG000B9Y6W2"
	Name     string       // e.g. "Apple Inc."
	Type     SecurityType // e.g. equity
}

func (s *Symbol) IsEmpty() bool {
	return !s.HasExchange() && !s.HasTicker() && !s.HasISIN() && !s.HasWKN() && !s.HasFIGI()
}

func (s *Symbol) HasExchange() bool { return s.Exchange != "" }

func (s *Symbol) HasTicker() bool { return s.Ticker != "" }

func (s *Symbol) HasISIN() bool { return s.ISIN != "" }

func (s *Symbol) HasWKN() bool { return s.WKN != "" }

func (s *Symbol) HasFIGI() bool { return s.FIGI != "" }

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

func IsISIN(s string) bool {
	return (&isinValidator{}).Validate(strings.ToUpper(s))
}

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

func IsFIGI(s string) bool {
	return (&figiValidator{}).Validate(strings.ToUpper(s))
}

// SymbolResolver searches for financial instruments.
type SymbolResolver interface {
	APIProvider

	// SearchSymbol looks up symbols matching the given free-text query (name, ticker, ISIN, WKN, etc.).
	SearchSymbol(ctx context.Context, query string) ([]Symbol, error)
}
