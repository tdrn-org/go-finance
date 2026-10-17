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
	"bytes"
	"compress/flate"
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrListingSearchRestricted indicates a search request could not be
	// fullfilled, as the provider does not support the given kind of query
	// (e.g. only ID searches are supported, but no free-text searches).
	ErrListingSearchRestricted error = errors.New("listing search restricted")
	// ErrListingProviderMismatch indicates a [Listing] cannot be processed
	// by the current provider, as it has been issued by another provider.
	ErrListingProviderMismatch error = errors.New("listing provider mismatch")
	// ErrQuoteNotAvailable indicates a provider is not able to
	// provide a quote for a given [Listing]. This is a
	// permanent error.
	ErrQuoteNotAvailable error = errors.New("quote not available")
	// ErrHistoryNotAvailable indicates a provider is not able to
	// provide a price history for a given [Listing]. This is a
	// permanent error.
	ErrHistoryNotAvailable error = errors.New("history not available")
)

// ListingAttributeKey is the key type for listing attributes.
type ListingAttributeKey string

const (
	// ListingAttributeKeySymbol attribute gives a provider specific symbol.
	// Such symbols are strictly provider specific and have no meaning at
	// all for other providers than the one issueing the listing.
	ListingAttributeKeySymbol ListingAttributeKey = "symbol"
	// ListingAttributeKeyTicker attribute gives the Ticker symbol for the
	// instrument represented by a listing
	// (https://en.wikipedia.org/wiki/Ticker_symbol).
	ListingAttributeKeyTicker ListingAttributeKey = "ticker"
	// ListingAttributeKeyISIN attribute gives the ISIN for the
	// instrument represented by a listing
	// (https://en.wikipedia.org/wiki/International_Securities_Identification_Number).
	ListingAttributeKeyISIN ListingAttributeKey = "isin"
	// ListingAttributeKeyWKN attribute gives the WKN for the
	// instrument represented by a listing
	// (https://en.wikipedia.org/wiki/Wertpapierkennnummer).
	ListingAttributeKeyWKN ListingAttributeKey = "wkn"
	// ListingAttributeKeyExchange attribute gives a provider specific trading
	// platform name the listing is traded at. Such names are strictly provider
	// specific and have no meaning at all for other providers than the one
	// issueing the listing.
	ListingAttributeKeyExchange ListingAttributeKey = "exchange"
	// ListingAttributeKeyMIC attribute gives the MIC code for the trading
	// platform the listing is traded at
	// (https://en.wikipedia.org/wiki/Market_Identifier_Code).
	ListingAttributeKeyMIC ListingAttributeKey = "mic"
)

func validateListingAttribute(key ListingAttributeKey, value string) error {
	if value == "" {
		return fmt.Errorf("empty attribute %s", key)
	}
	switch key {
	case ListingAttributeKeySymbol:
		// No special validation
	case ListingAttributeKeyTicker:
		if !IsTicker(value) {
			return fmt.Errorf("invalid Ticker attribute: %q", value)
		}
	case ListingAttributeKeyISIN:
		if !IsISIN(value) {
			return fmt.Errorf("invalid ISIN attribute: %q", value)
		}
	case ListingAttributeKeyWKN:
		if !IsWKN(value) {
			return fmt.Errorf("invalid WKN attribute: %q", value)
		}
	case ListingAttributeKeyExchange:
		// No special validation
	case ListingAttributeKeyMIC:
		if !IsMIC(value) {
			return fmt.Errorf("invalid MIC attribute: %q", value)
		}
	default:
		return fmt.Errorf("unrecognized attribute key: %q", key)
	}
	return nil
}

// ListingAttributeKeys contains all defined listing attribute keys.
var ListingAttributeKeys []ListingAttributeKey = []ListingAttributeKey{
	ListingAttributeKeySymbol,
	ListingAttributeKeyTicker,
	ListingAttributeKeyISIN,
	ListingAttributeKeyWKN,
	ListingAttributeKeyExchange,
	ListingAttributeKeyMIC,
}

// Listing is an opaque representation of a concrete listing
// of an financial instrument at an specific exchange. A Listing
// is issued by a specific provider and can only handled by the
// issueing provider.
//
// A Listing contains the following informaiton:
//   - Provider: The provider issueing this Listing. While a
//     Listing can be inspected from the outside (e.g.
//     [Listing.InstrumentName]), actual operations (e.g.
//     [ListingProvider.QueryQuote]) can only be peformed
//     by the issueing provider. [ErrListingProviderMismatch] indicates
//     indicates a failure in this regard.
//   - InstrumentName: The name of the financial instrument this
//     Listing is related to.
//   - InstrumentType: The type of the financial instrument this
//     Listing is related to.
//   - Attributes: A provider specific set of listing attributes,
//     used by the provider to identify the listing (see [ListingAttributeKey]).
//   - Currency: The currency used for this listing. This is an optional
//     information.
type Listing struct {
	data *listingData
	id   string
	s    string
}

type listingData struct {
	Provider       string                         `json:"provider"`
	IDKeys         []ListingAttributeKey          `json:"id_keys"`
	InstrumentName string                         `json:"instrument_name"`
	InstrumentType InstrumentType                 `json:"instrument_type"`
	Attributes     map[ListingAttributeKey]string `json:"attributes"`
	Currency       *Currency                      `json:"currency,omitempty"`
}

var listingTokenDict []byte = []byte(`
"provider":
,"id_keys":
,"instrument_name":
,"insturment_type":"equity"
,"insturment_type":"etf"
,"insturment_type":"fund"
,"insturment_type":"bond"
,"insturment_type":"index"
,"insturment_type":"forex"
,"insturment_type":"crypto"
,"insturment_type":"option"
,"insturment_type":"future"
,"insturment_type":"certificate"
,"attributes":
,"currency":
"symbol"
"ticker"
"isin"
"wkn"
"exchange"
"mic"`)

func (d *listingData) marshalText() ([]byte, error) {
	encoded1, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	encoded2 := &bytes.Buffer{}
	encoder2, err := flate.NewWriterDict(encoded2, flate.BestCompression, listingTokenDict)
	if err != nil {
		return nil, err
	}
	_, err = encoder2.Write(encoded1)
	if err != nil {
		return nil, err
	}
	err = encoder2.Close()
	if err != nil {
		return nil, err
	}
	return []byte(base64.RawURLEncoding.EncodeToString(encoded2.Bytes())), nil
}

func (d *listingData) unmarshalText(text []byte) error {
	decoded2, err := base64.RawURLEncoding.DecodeString(string(text))
	if err != nil {
		return err
	}
	decoder1 := flate.NewReaderDict(bytes.NewReader(decoded2), listingTokenDict)
	decoded1, err := io.ReadAll(decoder1)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded1, d)
}

// Provider gets the provider who issued this [Listing].
func (l *Listing) Provider() string {
	return l.data.Provider
}

// ID gets the unique ID for this [Listing].
func (l *Listing) ID() string {
	return l.id
}

// CheckProvider checks whether the given provider name matches
// the issueing provider of this [Listing].
// If not matching, [ErrListingProviderMismatch] is returned.
func (l *Listing) CheckProvider(name string) error {
	if l.data.Provider != name {
		return ErrListingProviderMismatch
	}
	return nil
}

// Provider gets the instrument name for this [Listing].
func (l *Listing) InstrumentName() string {
	return l.data.InstrumentName
}

// Provider gets the instrument type for this [Listing].
func (l *Listing) InstrumentType() InstrumentType {
	return l.data.InstrumentType
}

// Attributes gets the given attribute value for this [Listing].
// The returned bool indicates whether the attribute is available at al.
func (l *Listing) Attribute(key ListingAttributeKey) (string, bool) {
	value, ok := l.data.Attributes[key]
	return value, ok
}

// MustAttribute gets the given attribute value for this [Listing].
// If the attribute is not set, an error is returned.
func (l *Listing) MustAttribute(key ListingAttributeKey) (string, error) {
	value, ok := l.data.Attributes[key]
	if !ok {
		return "", fmt.Errorf("attribute %s not set", key)
	}
	return value, nil
}

// Provider gets the currency type for this [Listing] (if available).
func (l *Listing) Currency() *Currency {
	if l.data.Currency == nil {
		return nil
	}
	currency := *l.data.Currency
	return &currency
}

func (l *Listing) reset() {
	buffer := &strings.Builder{}

	// Reset id
	buffer.WriteString(l.data.Provider)
	for _, idKey := range l.data.IDKeys {
		buffer.WriteRune(':')
		buffer.WriteString(string(idKey))
		buffer.WriteRune('=')
		buffer.WriteString(l.data.Attributes[idKey])
	}
	l.id = buffer.String()

	buffer.Reset()

	// Reset s
	buffer.WriteString(l.data.InstrumentName)
	buffer.WriteString(" (provider:")
	buffer.WriteString(string(l.data.Provider))
	buffer.WriteString(", type:")
	buffer.WriteString(string(l.data.InstrumentType))
	for _, key := range ListingAttributeKeys {
		value, ok := l.data.Attributes[key]
		if !ok {
			continue
		}
		buffer.WriteString(", ")
		buffer.WriteString(string(key))
		buffer.WriteRune(':')
		buffer.WriteString(value)
	}
	buffer.WriteRune(')')
	l.s = buffer.String()
}

func (l *Listing) String() string {
	return l.s
}

func (l Listing) Validate() (*Listing, error) {
	errs := make([]error, 0)
	if l.data.Provider == "" {
		errs = append(errs, errors.New("no listing provider"))
	}
	if len(l.data.IDKeys) == 0 {
		errs = append(errs, errors.New("no id keys"))
	}
	if l.data.InstrumentName == "" {
		errs = append(errs, errors.New("no listing instrument name"))
	}
	if len(l.data.Attributes) == 0 {
		errs = append(errs, errors.New("no listing attributes"))
	} else {
		for key, value := range l.data.Attributes {
			err := validateListingAttribute(key, value)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	err := errors.Join(errs...)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (l Listing) MustValidate() *Listing {
	_, err := l.Validate()
	if err != nil {
		panic(err)
	}
	return &l
}

func (l *Listing) MarshalJSON() ([]byte, error) {
	text, err := l.data.marshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

func (l *Listing) UnmarshalJSON(data []byte) error {
	l.data = &listingData{}
	l.reset()
	var text string
	err := json.Unmarshal(data, &text)
	if err != nil {
		return err
	}
	err = l.data.unmarshalText([]byte(text))
	if err != nil {
		return err
	}
	l.reset()
	return nil
}

func (l *Listing) Value() (driver.Value, error) {
	data, err := l.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (l *Listing) Scan(value any) error {
	l.data = &listingData{}
	l.reset()
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		return l.UnmarshalJSON([]byte(v))
	case []byte:
		return l.UnmarshalJSON(v)
	default:
		return fmt.Errorf("unsupported type for Listing: %T", value)
	}
}

// ListingBuilder implements the builder pattern to create
// a new [Listing].
type ListingBuilder struct {
	listing Listing
}

// NewListing
func NewListing(provider string, idKeys ...ListingAttributeKey) *ListingBuilder {
	return &ListingBuilder{
		listing: Listing{
			data: &listingData{
				Provider:   provider,
				IDKeys:     idKeys,
				Attributes: make(map[ListingAttributeKey]string),
			},
		},
	}
}

func (b *ListingBuilder) InstrumentName(instrumentName string) *ListingBuilder {
	b.listing.data.InstrumentName = instrumentName
	return b
}

func (b *ListingBuilder) InstrumentType(instrumentType InstrumentType) *ListingBuilder {
	b.listing.data.InstrumentType = instrumentType
	return b
}

func (b *ListingBuilder) Attribute(key ListingAttributeKey, value string) *ListingBuilder {
	b.listing.data.Attributes[key] = value
	return b
}

func (b *ListingBuilder) Currency(currency *Currency) *ListingBuilder {
	if currency != nil {
		tmp := *currency
		b.listing.data.Currency = &tmp
	} else {
		b.listing.data.Currency = nil
	}
	return b
}

func (b *ListingBuilder) Build() Listing {
	b.listing.reset()
	return b.listing
}

type ListingPrice struct {
	// Instrument the financial instrument this price is for.
	Listing Listing `json:"listing"`
	// Timestamp gives the point in time this price was current
	// according to the sourcing provider.
	Timestamp time.Time `json:"timestamp"`
	// Currency gives the currency for the given values.
	Currency Currency `json:"currency"`
	// Price gives the current price at the trading day identified by Timestamp.
	Price float64 `json:"price"`
	// Sources defines the provider this price has been
	// queried from.
	Source string `json:"source"`
	// SourceTimestamp defines the time this price has
	// been queried.
	SourceTimestamp time.Time `json:"source_timestamp"`
}

// String formats this Quote.
func (q *ListingPrice) String() string {
	buffer := &strings.Builder{}
	buffer.WriteString(q.Listing.String())
	buffer.WriteString("|timestamp:")
	buffer.WriteString(q.Timestamp.UTC().Format(time.RFC3339Nano))
	buffer.WriteString("|price:")
	buffer.WriteString(strconv.FormatFloat(q.Price, 'g', -1, 64))
	buffer.WriteString("|currency:")
	buffer.WriteString(string(q.Currency))
	buffer.WriteString("|source:")
	buffer.WriteString(q.Source)
	buffer.WriteString("|queried:")
	buffer.WriteString(q.SourceTimestamp.UTC().Format(time.RFC3339Nano))
	return buffer.String()
}

// Quote represents a single full price data point for a [Listing].
type Quote struct {
	ListingPrice
	// Open gives the open price at the trading day identified by Timestamp.
	Open float64 `json:"open"`
	// High gives the high price at the trading day identified by Timestamp.
	High float64 `json:"high"`
	// Low gives the low price at the trading day identified by Timestamp.
	Low float64 `json:"low"`
	// Close gives the close price of the day before the trading day identified by Timestamp.
	Close float64 `json:"close"`
	// Volume gives the order volume at the trading day identified by Timestamp.
	Volume int64 `json:"volume"`
}

type ListingProvider interface {
	APIProvider

	// SearchListings searches for listings matching the given query.
	SearchListings(ctx context.Context, query string) ([]Listing, error)

	// QueryQuote returns the latest quote for the given listing.
	QueryQuote(ctx context.Context, listing Listing) (*Quote, error)
}
