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
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func TestListingToken(t *testing.T) {
	listing1 := *finance.NewProviderListing("test", finance.ListingAttributeKeyTicker, finance.ListingAttributeKeyMIC).
		InstrumentName("Test instrument").
		InstrumentType(finance.InstrumentTypeEquity).
		Attribute(finance.ListingAttributeKeyTicker, "TCKR").
		Attribute(finance.ListingAttributeKeyMIC, "XMIC").
		Build().MustValidate()
	token, err := listing1.Token()
	require.NoError(t, err)
	listing2, err := finance.NewProviderListingFromToken(token)
	require.NoError(t, err)
	require.Equal(t, listing1.String(), listing2.String())
}
