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
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
)

var (
	// ErrNoIdentifiers indicates an Instrument has no identifier set.
	ErrNoIdentifiers error = errors.New("no identifiers")
	// ErrInvalidIdentifier indicates an Instrument has an invalid identifiery set.
	ErrInvalidIdentifier error = errors.New("invalid identifier")
	// ErrNoMIC indicates an Instrument has no MIC set.
	ErrNoMIC error = errors.New("no MIC")
	// ErrInstrumentSearchRestricted indicates a provider is not able to search
	// for a given query (e.g. only ISIN search is supported).
	ErrInstrumentSearchRestricted error = errors.New("instrument search restricted")
)

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

// InstrumentIdentifier type.
// There is set of well-known instrument identifiers (e.g. ISIN), but
// providers may define their own in case they using non-standardized
// identifiers.
type InstrumentIdentifier string

const (
	// Ticker
	InstrumentIdentifierTicker InstrumentIdentifier = "ticker"
	// ISIN (International Securities Identification Number)
	InstrumentIdentifierISIN InstrumentIdentifier = "isin"
	// WKN (Wertpapierkennnummer)
	InstrumentIdentifierWKN InstrumentIdentifier = "wkn"
	// FIGI (Financial Instrument Global Identifier)
	InstrumentIdentifierFIGI InstrumentIdentifier = "figi"
)

// Instrument represents a financial instrument.
type Instrument struct {
	// ID is the internal identifier of this instrument.
	ID string `json:"id"`
	// Identifiers contains one or more identifiers for this
	// instrument (e.g. Ticker, ISIN, FIGI). At least one identifier
	// must be defined for a valid Instrument.
	Identifiers map[InstrumentIdentifier]string `json:"identifiers"`
	// Name contains the display name of this instrument.
	// A non-empty name is required for a valid Instrument.
	Name string `json:"name"`
	// MIC contains the MIC code of the exchange on which this
	// instrument is traded. A MIC is required for a valid instrument.
	MIC string `json:"mic"`
	// Currency contains the trading currency of this instrument.
	// This information is optional and may only become available
	// after querying instrument details or retrieving a quote.
	Currency *Currency `json:"currency,omitempty"`
	// Type classifies this instrument.
	Type InstrumentType `json:"type"`
}

// NewInstrumentID generates a new Instrument ID.
func NewInstrumentID() string {
	return uuid.NewString()
}

// NewInstrument creates new Instrument only with ID set.
// All other attributes must be set by the caller to create
// a non-empty Instrument.
func NewInstrument() *Instrument {
	return &Instrument{
		ID:          NewInstrumentID(),
		Identifiers: make(map[InstrumentIdentifier]string),
	}
}

// Validate checks whether this Instrument is valid.
//
// An instrument is valid, if and only if:
//   - ID is a valid UUID (as set by [NewInstrument])
//   - Has at least one identifier set
//   - Has only valid identifiers set
//   - Has the MIC set
func (i *Instrument) Validate() error {
	errs := make([]error, 0)
	_, uuidErr := uuid.Parse(i.ID)
	if uuidErr != nil {
		errs = append(errs, uuidErr)
	}
	if len(i.Identifiers) == 0 {
		errs = append(errs, ErrNoIdentifiers)
	}
	for key, value := range i.Identifiers {
		if value == "" {
			errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidIdentifier, key))
		}
	}
	if i.MIC == "" {
		errs = append(errs, ErrNoMIC)
	}
	return errors.Join(errs...)
}

// HasIdentifier checks whether the given identifier type is set for
// this Instrument.
func (i *Instrument) HasIdentifier(instrumentIdentifier InstrumentIdentifier) bool {
	_, has := i.Identifiers[instrumentIdentifier]
	return has
}

// Identifier gets the given identifier type for this Instrument. The 2nd
// return value indicates, whether the identifier exists.
func (i *Instrument) Identifier(instrumentIdentifier InstrumentIdentifier) (string, bool) {
	identifier, has := i.Identifiers[instrumentIdentifier]
	return identifier, has
}

// Clone clones this Instrument.
func (i *Instrument) Clone() Instrument {
	return Instrument{
		ID:          i.ID,
		Identifiers: maps.Clone(i.Identifiers),
		Name:        i.Name,
		MIC:         i.MIC,
		Currency:    i.Currency,
		Type:        i.Type,
	}
}

// Merge merges the given Instrument with this one by enriching
// this Instrument's data. No data is replaced, only new data
// is added.
func (i *Instrument) Merge(o *Instrument) {
	for key, value := range o.Identifiers {
		_, exists := i.Identifiers[key]
		if exists {
			continue
		}
		i.Identifiers[key] = value
	}
	if i.Currency == nil {
		i.Currency = o.Currency
	}
}

// String formats this Instrument.
func (i *Instrument) String() string {
	keys := slices.Collect(maps.Keys(i.Identifiers))
	slices.Sort(keys)
	buffer := &strings.Builder{}
	for _, key := range keys {
		if buffer.Len() > 0 {
			buffer.WriteRune('|')
		}
		buffer.WriteString(string(key))
		buffer.WriteRune(':')
		buffer.WriteString(i.Identifiers[key])
	}
	buffer.WriteString("|mic:")
	buffer.WriteString(i.MIC)
	return buffer.String()
}

// InstrumentProvider searches and resolves instruments.
type InstrumentProvider interface {
	APIProvider
	// SearchInstruments searches for instruments matching the given query.
	SearchInstruments(ctx context.Context, query string) ([]Instrument, error)
	// ResolveInstruments resolves the given instruments by enriching their data.
	// See [Instrument.Merge].
	ResolveInstruments(ctx context.Context, instruments []Instrument) ([]Instrument, error)
}
