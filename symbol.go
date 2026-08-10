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

import "context"

// SecurityType classifies a financial instrument.
type SecurityType string

const (
	SecurityTypeEquity SecurityType = "equity"
	SecurityTypeETF    SecurityType = "etf"
)

// ISO 10383 Market Identifier Codes (MIC).
// Not exhaustive — extend as needed.
const (
	MIC_XNAS = "XNAS" // NASDAQ
	MIC_XNYS = "XNYS" // NYSE
	MIC_XETR = "XETR" // Xetra
	MIC_XSTU = "XSTU" // Stuttgart
	MIC_XBER = "XBER" // Berlin
)

// Symbol identifies a financial instrument.
// It is a composite of identifiers from different naming systems.
// Not all fields are guaranteed to be populated; a provider fills
// whichever identifiers it supports. Callers should check with
// the Has* methods before relying on a specific field.
type Symbol struct {
	Ticker   string       // e.g. "AAPL"
	Exchange string       // MIC-Code, e.g. "XNAS"
	Name     string       // e.g. "Apple Inc."
	ISIN     string       // e.g. "US0378331005"
	WKN      string       // e.g. "865985"
	FIGI     string       // e.g. "BBG000B9Y6W2"
	Currency Currency     // e.g. USD
	Type     SecurityType // e.g. equity
}

func (s *Symbol) HasTicker() bool { return s.Ticker != "" }

func (s *Symbol) HasExchange() bool { return s.Exchange != "" }

func (s *Symbol) HasISIN() bool { return s.ISIN != "" }

func (s *Symbol) HasWKN() bool { return s.WKN != "" }

func (s *Symbol) HasFIGI() bool { return s.FIGI != "" }

// ExchangeTicker returns a stable composite key for cache and map lookups,
// e.g. "XNAS:AAPL". Returns just the ticker if no exchange is set.
func (s *Symbol) ExchangeTicker() string {
	if s.Exchange != "" {
		return s.Exchange + ":" + s.Ticker
	}
	return s.Ticker
}

// SymbolResolver searches for financial instruments.
type SymbolResolver interface {
	APIProvider

	// Search looks up symbols matching the given free-text query (name, ticker, ISIN, WKN, etc.).
	Search(ctx context.Context, query string) ([]Symbol, error)

	// LookupByISIN returns the exact symbol for an ISIN, if known.
	LookupByISIN(ctx context.Context, isin string) (*Symbol, error)
}
