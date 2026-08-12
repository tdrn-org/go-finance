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

package openfigi

import (
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/openfigi/api"
)

var securityTypeMap map[string]string = map[string]string{
	"Common Stock": string(finance.SecurityTypeEquity),
}

func figiResultToSymbol(figiResult *api.FigiResult) *finance.Symbol {
	if figiResult == nil {
		return nil
	}
	symbol := &finance.Symbol{
		Ticker: ptrString(figiResult.Ticker),
		Name:   ptrString(figiResult.Name),
		FIGI:   ptrString(figiResult.Figi),
		Type:   finance.MapSecurityType(ptrString(figiResult.SecurityType), securityTypeMap),
	}
	if symbol.IsEmpty() {
		return nil
	}
	return symbol
}

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
