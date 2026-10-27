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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrEmptyListing indicates a [Listing] is empty is therefore invalid.
	ErrEmptyListing error = errors.New("empty listing")
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
	case ListingAttributeKeySymbol, ListingAttributeKeyExchange:
		// Symbols and exchange names are provider specific and have no standard
		// format, but they become segments of the listing ID. The ID separators
		// are therefore rejected: a separator inside a value would split one
		// segment into two and let a caller forge an identity.
		if strings.ContainsRune(value, ':') || strings.ContainsRune(value, '=') {
			return fmt.Errorf("invalid %s attribute: %q must not contain ':' or '='", key, value)
		}
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

// listingData is the internal representation of a [Listing] and, at the same
// time, its persisted form.
//
// The JSON tags below are the storage format written by [Listing.Value] and read
// by [Listing.Scan]. They are deliberately plain keys and values: a stored
// listing must stay readable with nothing but a JSON parser, because whoever
// inspects the database later has no access to this code. Renaming a tag
// therefore requires a migration, not just a refactoring.
type listingData struct {
	Provider       string                         `json:"provider"`
	IDKeys         []ListingAttributeKey          `json:"id_keys"`
	InstrumentName string                         `json:"instrument_name"`
	InstrumentType InstrumentType                 `json:"instrument_type"`
	Attributes     map[ListingAttributeKey]string `json:"attributes"`
	Currency       *Currency                      `json:"currency,omitempty"`
}

// ParseListingID reconstructs the identity carried by a listing ID.
//
// A listing ID has the form <provider>[:<key>=<value>]*, with one segment per id
// key in the order given to [NewListing], e.g.
// "twelvedata:ticker=AAPL:mic=XNGS".
//
// The returned listing carries the identity but no description: instrument name
// and type are not part of an ID, so [Listing.Validate] does not accept such a
// listing as complete. It is enough to identify an instrument, not to describe
// one - use it to resolve an ID against the store, not to render it.
func ParseListingID(id string) (Listing, error) {
	listing := Listing{data: &listingData{Attributes: make(map[ListingAttributeKey]string)}}
	segments := strings.Split(id, ":")
	listing.data.Provider = segments[0]
	if listing.data.Provider == "" {
		return listing, fmt.Errorf("invalid listing ID %q: no provider", id)
	}
	for _, segment := range segments[1:] {
		key, value, found := strings.Cut(segment, "=")
		if !found || key == "" {
			return listing, fmt.Errorf("invalid listing ID %q: malformed segment %q", id, segment)
		}
		listing.data.IDKeys = append(listing.data.IDKeys, ListingAttributeKey(key))
		listing.data.Attributes[ListingAttributeKey(key)] = value
	}
	if len(listing.data.IDKeys) == 0 {
		return listing, fmt.Errorf("invalid listing ID %q: no id attributes", id)
	}
	listing.reset()
	return listing, nil
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

// Provider gets the provider who issued this [Listing].
func (l *Listing) Provider() string {
	if l.data == nil {
		return ""
	}
	return l.data.Provider
}

// ID gets the unique ID for this [Listing].
func (l *Listing) ID() string {
	return l.id
}

// CheckProvider checks whether the given provider matches
// the issueing provider of this [Listing].
// If not matching, [ErrListingProviderMismatch] is returned.
func (l *Listing) CheckProvider(provider ListingProvider) error {
	return l.CheckProviderName(provider.ProviderName())
}

// CheckProviderName checks whether the given provider name matches
// the issueing provider's name of this [Listing].
// If not matching, [ErrListingProviderMismatch] is returned.
func (l *Listing) CheckProviderName(name string) error {
	if l.data == nil {
		return ErrEmptyListing
	}
	if l.data.Provider != name {
		return ErrListingProviderMismatch
	}
	return nil
}

// Provider gets the instrument name for this [Listing].
func (l *Listing) InstrumentName() string {
	if l.data == nil {
		return ""
	}
	return l.data.InstrumentName
}

// Provider gets the instrument type for this [Listing].
func (l *Listing) InstrumentType() InstrumentType {
	if l.data == nil {
		return InstrumentTypeUnknown
	}
	return l.data.InstrumentType
}

// Attributes gets the given attribute value for this [Listing].
// The returned bool indicates whether the attribute is available at al.
func (l *Listing) Attribute(key ListingAttributeKey) (string, bool) {
	if l.data == nil {
		return "", false
	}
	value, ok := l.data.Attributes[key]
	return value, ok
}

// MustAttribute gets the given attribute value for this [Listing].
// If the attribute is not set, an error is returned.
func (l *Listing) MustAttribute(key ListingAttributeKey) (string, error) {
	if l.data == nil {
		return "", ErrEmptyListing
	}
	value, ok := l.data.Attributes[key]
	if !ok {
		return "", fmt.Errorf("attribute %s not set", key)
	}
	return value, nil
}

// Provider gets the currency type for this [Listing] (if available).
func (l *Listing) Currency() *Currency {
	if l.data == nil {
		return nil
	}
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

// String renders the listing for logs and error messages.
//
// A value receiver on purpose: a Listing is passed around by value (position
// fields, response payloads), and a pointer-only Stringer would silently fall
// back to the raw struct for those.
func (l Listing) String() string {
	return l.s
}

func (l Listing) Validate() (*Listing, error) {
	if l.data == nil {
		return &l, ErrEmptyListing
	}
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
	// Every id key must carry a value. The ID is built from exactly those
	// attributes, so a missing one yields an ID with an empty segment - a
	// well-formed key for a different instrument, which would silently collide
	// with it in the store. An id key that is not a known attribute key can be
	// neither validated nor looked up and is rejected for the same reason.
	seenIDKeys := make(map[ListingAttributeKey]bool, len(l.data.IDKeys))
	for _, idKey := range l.data.IDKeys {
		if seenIDKeys[idKey] {
			errs = append(errs, fmt.Errorf("duplicate id key %s", idKey))
			continue
		}
		seenIDKeys[idKey] = true
		if value, ok := l.data.Attributes[idKey]; !ok || value == "" {
			errs = append(errs, fmt.Errorf("no value for id attribute %s", idKey))
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

// MarshalJSON encodes the listing as its ID.
//
// The wire form carries the identity only - it is the key a caller hands back to
// reach the same instrument again. Instrument name and type are deliberately not
// part of it: they are description, not identity, and the store is the only
// trustworthy source for them. Use [ParseListingID] to read the identity back.
//
// A value receiver on purpose: a Listing embedded in a struct that is marshalled
// by value is not addressable, and encoding/json silently ignores a pointer-only
// Marshaler in that case - the listing would render as an empty object on the
// wire instead of an ID.
//
// Rendering never fails on a broken listing. A marshal error aborts the whole
// payload, so one unusable instrument would hide every other field from the
// client; an empty string is visible, and [ParseListingID] rejects it on the way
// back. Persisting is the opposite - see [Listing.Value], which refuses to store
// what cannot be identified.
func (l Listing) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.id)
}

// UnmarshalJSON reads a listing ID, see [ParseListingID].
//
// The result carries the identity only and therefore does not pass
// [Listing.Validate]. Resolve it against the store to get the full listing.
func (l *Listing) UnmarshalJSON(data []byte) error {
	var id string
	err := json.Unmarshal(data, &id)
	if err != nil {
		return err
	}
	parsed, err := ParseListingID(id)
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

// Value returns the persisted form of the listing: its plain JSON representation.
//
// Storage must stand on its own. A stored listing has to remain readable after
// the wire format or any encoding detail changed, and it has to be readable by
// someone who has nothing but a database client - hence plain JSON and not the
// ID, which cannot be rendered into a description.
//
// A value receiver on purpose: database/sql only honours driver.Valuer when the
// type implements it, and a Listing passed by value would otherwise reach the
// driver as a struct.
func (l Listing) Value() (driver.Value, error) {
	if l.data == nil {
		return nil, ErrEmptyListing
	}
	data, err := json.Marshal(l.data)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// Scan reads the persisted form written by [Listing.Value].
func (l *Listing) Scan(value any) error {
	if value == nil {
		*l = Listing{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("unsupported type for Listing: %T", value)
	}
	scanned := listingData{}
	err := json.Unmarshal(data, &scanned)
	if err != nil {
		return err
	}
	if scanned.Attributes == nil {
		scanned.Attributes = make(map[ListingAttributeKey]string)
	}
	*l = Listing{data: &scanned}
	l.reset()
	return nil
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

// String formats this ListingPrice.
func (p *ListingPrice) String() string {
	buffer := &strings.Builder{}
	buffer.WriteString(p.Listing.ID())
	buffer.WriteString("|timestamp:")
	buffer.WriteString(p.Timestamp.UTC().Format(time.RFC3339Nano))
	buffer.WriteString("|price:")
	buffer.WriteString(strconv.FormatFloat(p.Price, 'g', -1, 64))
	buffer.WriteString("|currency:")
	buffer.WriteString(string(p.Currency))
	buffer.WriteString("|source:")
	buffer.WriteString(p.Source)
	buffer.WriteString("|queried:")
	buffer.WriteString(p.SourceTimestamp.UTC().Format(time.RFC3339Nano))
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
	// PreviousClose gives the close price of the day before the trading day identified by Timestamp.
	PreviousClose float64 `json:"close"`
	// Volume gives the order volume at the trading day identified by Timestamp.
	Volume int64 `json:"volume"`
}

// String formats this Quote.
func (q *Quote) String() string {
	buffer := &strings.Builder{}
	buffer.WriteString(q.Listing.ID())
	buffer.WriteString("|timestamp:")
	buffer.WriteString(q.Timestamp.UTC().Format(time.RFC3339Nano))
	buffer.WriteString("|price:")
	buffer.WriteString(strconv.FormatFloat(q.Price, 'g', -1, 64))
	buffer.WriteString("|open:")
	buffer.WriteString(strconv.FormatFloat(q.Open, 'g', -1, 64))
	buffer.WriteString("|high:")
	buffer.WriteString(strconv.FormatFloat(q.High, 'g', -1, 64))
	buffer.WriteString("|low:")
	buffer.WriteString(strconv.FormatFloat(q.Low, 'g', -1, 64))
	buffer.WriteString("|previous_close:")
	buffer.WriteString(strconv.FormatFloat(q.PreviousClose, 'g', -1, 64))
	buffer.WriteString("|currency:")
	buffer.WriteString(string(q.Currency))
	buffer.WriteString("|volume:")
	buffer.WriteString(strconv.FormatInt(q.Volume, 10))
	buffer.WriteString("|source:")
	buffer.WriteString(q.Source)
	buffer.WriteString("|queried:")
	buffer.WriteString(q.SourceTimestamp.UTC().Format(time.RFC3339Nano))
	return buffer.String()
}

type ListingProvider interface {
	APIProvider

	// SearchListings searches for listings matching the given query.
	SearchListings(ctx context.Context, query string) ([]Listing, error)

	// QueryQuote returns the latest quote for the given listing.
	QueryQuote(ctx context.Context, listing Listing) (*Quote, error)

	// QueryQuoteHistory returns the quote history for the given listing and given time range.
	QueryQuoteHistory(ctx context.Context, listing Listing, from, to time.Time) ([]Quote, error)
}
