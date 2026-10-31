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

package finance_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

type listingStruct struct {
	Listing finance.Listing `json:"listing"`
}

// TestJSONMarshaling covers the wire form: a listing is transferred as its ID.
//
// The ID carries the identity, not the description. A caller that hands the ID
// back reaches the same instrument; what that instrument is called is resolved
// from the store, which is the only trustworthy source for it.
func TestJSONMarshaling(t *testing.T) {
	s1 := &listingStruct{Listing: *testListing()}
	encoded, err := json.Marshal(s1)
	require.NoError(t, err)
	require.Equal(t, `{"listing":"test:ticker=TCKR:mic=XMIC"}`, string(encoded))
	s2 := &listingStruct{}
	err = json.Unmarshal(encoded, s2)
	require.NoError(t, err)
	// The ID is the identity carrier: it is used as the database key and as the
	// map key when quotes are matched against positions. A round trip that only
	// preserved the attributes would leave a decoded listing with a foreign
	// identity.
	require.Equal(t, s1.Listing.ID(), s2.Listing.ID())
	require.Equal(t, s1.Listing.Provider(), s2.Listing.Provider())
	for _, key := range finance.ListingAttributeKeys {
		v1, ok1 := s1.Listing.Attribute(key)
		v2, ok2 := s2.Listing.Attribute(key)
		require.Equal(t, ok1, ok2, "attribute presence differs for %s", key)
		require.Equal(t, v1, v2, "attribute value differs for %s", key)
	}
	// Description is not part of the wire form and therefore not restorable from
	// it. This is intentional, see [finance.Listing.MarshalJSON].
	require.Empty(t, s2.Listing.InstrumentName())
	require.Equal(t, finance.InstrumentTypeUnknown, s2.Listing.InstrumentType())
	// An identity-only listing identifies an instrument, it does not describe
	// one: it must never reach the store as a complete listing.
	_, err = s2.Listing.Validate()
	require.Error(t, err)
}

// TestDatabaseRoundTrip covers the persisted form, which has to be lossless.
func TestDatabaseRoundTrip(t *testing.T) {
	s1 := testListing()
	value, err := s1.Value()
	require.NoError(t, err)
	// Storage has to be readable without this code: plain JSON, not an ID.
	text, ok := value.(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(text, "{"), "stored listing must be plain JSON, got %q", text)
	require.Contains(t, text, `"instrument_name":"Test instrument"`)

	s2 := &finance.Listing{}
	err = s2.Scan(value)
	require.NoError(t, err)
	require.Equal(t, s1.String(), s2.String())
	require.Equal(t, s1.ID(), s2.ID())
	// Provider, instrument name and type are stored twice - inside the JSON and
	// in their own columns; both representations must agree.
	require.Equal(t, s1.Provider(), s2.Provider())
	require.Equal(t, s1.InstrumentName(), s2.InstrumentName())
	require.Equal(t, s1.InstrumentType(), s2.InstrumentType())
	for _, key := range finance.ListingAttributeKeys {
		v1, ok1 := s1.Attribute(key)
		v2, ok2 := s2.Attribute(key)
		require.Equal(t, ok1, ok2, "attribute presence differs for %s", key)
		require.Equal(t, v1, v2, "attribute value differs for %s", key)
	}
	// The persisted form is complete: it passes validation on its own.
	_, err = s2.Validate()
	require.NoError(t, err)
}

// TestScanNull covers a NULL column, which simply is an unset listing.
func TestScanNull(t *testing.T) {
	s := &finance.Listing{}
	require.NoError(t, s.Scan(nil))
	require.Empty(t, s.ID())
	require.Empty(t, s.Provider())
	_, err := s.Validate()
	require.ErrorIs(t, err, finance.ErrEmptyListing)
}

// TestEncodingIsDeterministic guards the identity carrier: the same listing must
// always produce the same ID, otherwise clients would see the identity of an
// unchanged instrument change under them.
func TestEncodingIsDeterministic(t *testing.T) {
	s := testListing()
	first, err := json.Marshal(s)
	require.NoError(t, err)
	for i := 0; i < 32; i++ {
		again, err := json.Marshal(s)
		require.NoError(t, err)
		require.Equal(t, string(first), string(again))
	}
}

// TestValidateRequiresAllIDKeys covers the identity invariant behind the ID: a
// listing whose id key carries no value would produce an ID with an empty
// segment, which is a well-formed key for a different instrument.
func TestValidateRequiresAllIDKeys(t *testing.T) {
	incomplete := finance.NewListing("test", finance.ListingAttributeKeyTicker, finance.ListingAttributeKeyMIC).
		InstrumentName("Test instrument").
		InstrumentType(finance.InstrumentTypeEquity).
		Attribute(finance.ListingAttributeKeyTicker, "TCKR").
		Build()
	_, err := incomplete.Validate()
	require.ErrorContains(t, err, "unset id attribute mic")
}

// TestParseListingID covers reading an ID back into an identity.
func TestParseListingID(t *testing.T) {
	parsed, err := finance.ParseListingID("test:ticker=TCKR:mic=XMIC")
	require.NoError(t, err)
	require.Equal(t, "test:ticker=TCKR:mic=XMIC", parsed.ID())
	require.Equal(t, "test", parsed.Provider())
	ticker, ok := parsed.Attribute(finance.ListingAttributeKeyTicker)
	require.True(t, ok)
	require.Equal(t, "TCKR", ticker)

	for _, invalid := range []string{
		"",
		"test",
		"test:ticker",
		"test:=TCKR",
		":ticker=TCKR",
		"test:ticker=TCKR:mic",
	} {
		_, err := finance.ParseListingID(invalid)
		require.Error(t, err, "expected %q to be rejected", invalid)
	}
}

// TestSymbolCannotForgeIdentity covers the ID as an attack surface: symbols and
// exchange names have no fixed format, but they become ID segments, so they must
// not be able to carry the segment separators.
func TestSymbolCannotForgeIdentity(t *testing.T) {
	for _, value := range []string{"A:B", "A=B", "A=B:mic=XMIC"} {
		forged := finance.NewListing("test", finance.ListingAttributeKeySymbol).
			InstrumentName("Test instrument").
			InstrumentType(finance.InstrumentTypeEquity).
			Attribute(finance.ListingAttributeKeySymbol, value).
			Build()
		_, err := forged.Validate()
		require.Error(t, err, "expected symbol %q to be rejected", value)
	}
}

// TestMarshalUnaddressableValue covers the case that makes a value receiver
// necessary: a listing embedded in a struct that is marshalled by value is not
// addressable, and encoding/json silently ignores a pointer-only Marshaler there.
// All fields of a Listing are unexported, so the fallback renders an empty object
// - the identity would vanish from the payload without any error.
func TestMarshalUnaddressableValue(t *testing.T) {
	s := testListing()
	encoded, err := json.Marshal(struct {
		Listing finance.Listing `json:"listing"`
	}{Listing: *s})
	require.NoError(t, err)
	require.Equal(t, `{"listing":"test:ticker=TCKR:mic=XMIC"}`, string(encoded))

	// The same through a slice, which is the shape the API actually returns.
	encoded, err = json.Marshal([]finance.Listing{*s})
	require.NoError(t, err)
	require.Equal(t, `["test:ticker=TCKR:mic=XMIC"]`, string(encoded))
}

// TestValueImplementsValuer covers the storage side of the same problem: a
// Listing passed by value must satisfy driver.Valuer, otherwise database/sql
// hands the struct itself to the driver.
func TestValueImplementsValuer(t *testing.T) {
	var valuer driver.Valuer = *testListing()
	value, err := valuer.Value()
	require.NoError(t, err)
	require.Contains(t, value.(string), `"instrument_name":"Test instrument"`)

	var scanner sql.Scanner = &finance.Listing{}
	require.NoError(t, scanner.Scan(value))
}

// TestStringImplementsStringer covers logs and error messages: a Listing value
// has to render as a description, not as its internal struct.
func TestStringImplementsStringer(t *testing.T) {
	var stringer fmt.Stringer = *testListing()
	require.Equal(t, "Test instrument (provider:test, type:equity, ticker:TCKR, mic:XMIC)", stringer.String())
}

// TestMarshalBrokenListingRendersEmpty pins the difference between rendering and
// storing: marshalling must not fail, because one unusable instrument would
// otherwise hide the whole payload. The empty ID is visible and rejected by
// ParseListingID on the way back.
func TestMarshalBrokenListingRendersEmpty(t *testing.T) {
	encoded, err := json.Marshal(struct {
		Listing finance.Listing `json:"listing"`
	}{})
	require.NoError(t, err)
	require.Equal(t, `{"listing":""}`, string(encoded))

	_, err = finance.ParseListingID("")
	require.Error(t, err)

	// Storing is strict, unlike rendering.
	_, err = finance.Listing{}.Value()
	require.ErrorIs(t, err, finance.ErrEmptyListing)
}

func testListing() *finance.Listing {
	return finance.NewListing("test", finance.ListingAttributeKeyTicker, finance.ListingAttributeKeyMIC).
		InstrumentName("Test instrument").
		InstrumentType(finance.InstrumentTypeEquity).
		Attribute(finance.ListingAttributeKeyTicker, "TCKR").
		Attribute(finance.ListingAttributeKeyMIC, "XMIC").
		Build().MustValidate()
}
