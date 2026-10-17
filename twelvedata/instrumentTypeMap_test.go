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

package twelvedata

import (
	"slices"
	"testing"

	"github.com/tdrn-org/go-finance"
)

// observedInstrumentTypes are the raw "instrument_type" values the Twelve Data
// symbol_search API returned for real queries (CoreWeave, Tesla, Vanguard, SPY,
// Realty Income).
//
// A missing entry does not fail loudly — the listing simply carries no instrument
// type, and include_instrument_types cannot match it either. Hence this test.
var observedInstrumentTypes []string = []string{
	"Common Stock",
	"American Depositary Receipt",
	"Depositary Receipt",
	"REIT",
	"ETF",
	"Mutual Fund",
	"Bond Fund",
	"Closed-end Fund",
	"Bond",
	"Index",
	"Forex",
	"Digital Currency",
	"Cryptocurrency",
	"Option",
	"Warrant",
	"Future",
	"Certificate",
}

func TestInstrumentTypeMapCoversObservedTypes(t *testing.T) {
	for _, rawType := range observedInstrumentTypes {
		if _, ok := instrumentTypeMap[rawType]; !ok {
			t.Errorf("raw instrument type %q is not mapped; listings of this type would have no instrument type", rawType)
		}
	}
}

func TestInstrumentTypeMapValuesAreKnownTypes(t *testing.T) {
	known := []finance.InstrumentType{
		finance.InstrumentTypeEquity,
		finance.InstrumentTypeETF,
		finance.InstrumentTypeFund,
		finance.InstrumentTypeBond,
		finance.InstrumentTypeIndex,
		finance.InstrumentTypeForex,
		finance.InstrumentTypeCrypto,
		finance.InstrumentTypeOption,
		finance.InstrumentTypeFuture,
		finance.InstrumentTypeCertificate,
	}
	for rawType, instrumentType := range instrumentTypeMap {
		if instrumentType == finance.InstrumentTypeUnknown {
			t.Errorf("raw instrument type %q maps to the unknown type", rawType)
			continue
		}
		if !slices.Contains(known, instrumentType) {
			t.Errorf("raw instrument type %q maps to %q, which is not a defined finance.InstrumentType", rawType, instrumentType)
		}
	}
}
