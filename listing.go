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
	"strings"
)

var (
	// ErrListingSearchRestricted indicates a search request could not be
	// fullfilled, as the provider does not support the given kind of query
	// (e.g. only ID searches are supported, but no free-text searches).
	ErrListingSearchRestricted error = errors.New("listing search restricted")
	// ErrListingProviderMismatch indicates a listing cannot be processed
	// by the current provider, as the listing has been issued by a different
	// provider.
	ErrListingProviderMismatch error = errors.New("listing provider mismatch")
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

// ProviderListing is an opaque representation of a concrete listing
// of an financial instrument at an specific exchange. A ProviderListing
// is issued by a specific provider and can only handled by the
// issueing provider.
//
// A ProviderListing contains the following informaiton:
//   - Provider: The provider issueing this ProviderListing. While a
//     ProviderListing can be inspected from the outside (e.g.
//     [ProviderListing.InstrumentName]), actual operations (e.g.
//     [ListingProvider.QueryQuote]) can only be peformed
//     by the issueing provider. [ErrListingProviderMismatch] indicates
//     indicates a failure in this regard.
//   - InstrumentName: The name of the financial instrument this
//     ProviderListing is related to.
//   - InstrumentType: The type of the financial instrument this
//     ProviderListing is related to.
//   - Attributes: A provider specific set of listing attributes,
//     used by the provider to identify the listing (see [ListingAttributeKey]).
//   - Currency: The currency used for this listing. This is an optional
//     information.
type ProviderListing struct {
	data *providerListingData
	id   string
	s    string
}

type providerListingData struct {
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

func (d *providerListingData) marshalText() ([]byte, error) {
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

func (d *providerListingData) unmarshalText(text []byte) error {
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

// Provider gets the provider who issued this [ProviderListing].
func (l *ProviderListing) Provider() string {
	return l.data.Provider
}

// ID gets the unique ID for this [ProviderListing].
func (l *ProviderListing) ID() string {
	return l.id
}

// CheckProvider checks whether the given provider name matches
// the issueing provider of this [ProviderListing].
// If not matching, [ErrListingProviderMismatch] is returned.
func (l *ProviderListing) CheckProvider(name string) error {
	if l.data.Provider != name {
		return ErrListingProviderMismatch
	}
	return nil
}

// Provider gets the instrument name for this [ProviderListing].
func (l *ProviderListing) InstrumentName() string {
	return l.data.InstrumentName
}

// Provider gets the instrument type for this [ProviderListing].
func (l *ProviderListing) InstrumentType() InstrumentType {
	return l.data.InstrumentType
}

// Attributes gets the given attribute value for this [ProviderListing].
// The returned bool indicates whether the attribute is available at al.
func (l *ProviderListing) Attribute(key ListingAttributeKey) (string, bool) {
	value, ok := l.data.Attributes[key]
	return value, ok
}

// MustAttribute gets the given attribute value for this [ProviderListing].
// If the attribute is not set, an error is returned.
func (l *ProviderListing) MustAttribute(key ListingAttributeKey) (string, error) {
	value, ok := l.data.Attributes[key]
	if !ok {
		return "", fmt.Errorf("attribute %s not set", key)
	}
	return value, nil
}

// Provider gets the currency type for this [ProviderListing] (if available).
func (l *ProviderListing) Currency() *Currency {
	if l.data.Currency == nil {
		return nil
	}
	currency := *l.data.Currency
	return &currency
}

func (l *ProviderListing) reset() {
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
	buffer.WriteString("provider:")
	buffer.WriteString(l.data.Provider)
	buffer.WriteString("|instrument_name:")
	buffer.WriteString(l.data.InstrumentName)
	buffer.WriteString("|instrument_type:")
	buffer.WriteString(string(l.data.InstrumentType))
	for _, key := range ListingAttributeKeys {
		value, ok := l.data.Attributes[key]
		if !ok {
			continue
		}
		buffer.WriteRune('|')
		buffer.WriteString(string(key))
		buffer.WriteRune(':')
		buffer.WriteString(value)
	}
	buffer.WriteString("|currency:")
	if l.data.Currency != nil {
		buffer.WriteString(string(*l.data.Currency))
	} else {
		buffer.WriteRune('-')
	}
	l.s = buffer.String()
}

func (l *ProviderListing) String() string {
	return l.s
}

func (l ProviderListing) Validate() (*ProviderListing, error) {
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

func (l ProviderListing) MustValidate() *ProviderListing {
	_, err := l.Validate()
	if err != nil {
		panic(err)
	}
	return &l
}

func (l *ProviderListing) MarshalJSON() ([]byte, error) {
	text, err := l.data.marshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

func (l *ProviderListing) UnmarshalJSON(data []byte) error {
	l.data = &providerListingData{}
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

func (l *ProviderListing) Value() (driver.Value, error) {
	data, err := l.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (l *ProviderListing) Scan(value any) error {
	l.data = &providerListingData{}
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
		return fmt.Errorf("unsupported type for ProviderListing: %T", value)
	}
}

// ProviderListingBuilder implements the builder pattern to create
// a new ProviderListing.
type ProviderListingBuilder struct {
	listing ProviderListing
}

// NewProviderListing
func NewProviderListing(provider string, idKeys ...ListingAttributeKey) *ProviderListingBuilder {
	return &ProviderListingBuilder{
		listing: ProviderListing{
			data: &providerListingData{
				Provider:   provider,
				IDKeys:     idKeys,
				Attributes: make(map[ListingAttributeKey]string),
			},
		},
	}
}

func (b *ProviderListingBuilder) InstrumentName(instrumentName string) *ProviderListingBuilder {
	b.listing.data.InstrumentName = instrumentName
	return b
}

func (b *ProviderListingBuilder) InstrumentType(instrumentType InstrumentType) *ProviderListingBuilder {
	b.listing.data.InstrumentType = instrumentType
	return b
}

func (b *ProviderListingBuilder) Attribute(key ListingAttributeKey, value string) *ProviderListingBuilder {
	b.listing.data.Attributes[key] = value
	return b
}

func (b *ProviderListingBuilder) Currency(currency *Currency) *ProviderListingBuilder {
	if currency != nil {
		tmp := *currency
		b.listing.data.Currency = &tmp
	} else {
		b.listing.data.Currency = nil
	}
	return b
}

func (b *ProviderListingBuilder) Build() ProviderListing {
	b.listing.reset()
	return b.listing
}

type ListingProvider interface {
	APIProvider

	// SearchListings searches for listings matching the given query.
	SearchListings(ctx context.Context, query string) ([]ProviderListing, error)

	// QueryQuote returns the latest quote for the given listing.
	QueryQuote(ctx context.Context, listing ProviderListing) (*Quote, error)
}
