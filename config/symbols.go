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

package config

import (
	"fmt"
	"log/slog"

	"github.com/tdrn-org/go-finance/alphavantage"
	"github.com/tdrn-org/go-finance/consorsbank"
	"github.com/tdrn-org/go-finance/openfigi"
	"github.com/tdrn-org/go-finance/twelvedata"
)

type SymbolsConfig struct {
	ProviderNames []SymbolsProviderName `toml:"providers"`
	Cooldown      DurationSpec          `toml:"cooldown"`
	CacheTTL      DurationSpec          `toml:"cache_ttl"`
}

type SymbolsProviderName string

const (
	SymbolsProviderNameAlphaVantage SymbolsProviderName = SymbolsProviderName(alphavantage.Name)
	SymbolsProviderNameConsorsbank  SymbolsProviderName = SymbolsProviderName(consorsbank.Name)
	SymbolsProviderNameOpenFIGI     SymbolsProviderName = SymbolsProviderName(openfigi.Name)
	SymbolsProviderNameTwelveData   SymbolsProviderName = SymbolsProviderName(twelvedata.Name)
)

var knownSymbolsProviderNames map[string]SymbolsProviderName = map[string]SymbolsProviderName{
	string(SymbolsProviderNameAlphaVantage): SymbolsProviderNameAlphaVantage,
	string(SymbolsProviderNameConsorsbank):  SymbolsProviderNameConsorsbank,
	string(SymbolsProviderNameOpenFIGI):     SymbolsProviderNameOpenFIGI,
	string(SymbolsProviderNameTwelveData):   SymbolsProviderNameTwelveData,
}

func (n *SymbolsProviderName) Value() string {
	for value, providerName := range knownSymbolsProviderNames {
		if *n == providerName {
			return value
		}
	}
	slog.Warn("unexpected Symbols provider name", slog.Any("n", *n))
	return ""
}

func (n *SymbolsProviderName) MarshalTOML() ([]byte, error) {
	return []byte(`"` + n.Value() + `"`), nil
}

func (n *SymbolsProviderName) UnmarshalTOML(value any) error {
	providerNameString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected Symbols provider name type %v", value)
	}
	providerName, ok := knownSymbolsProviderNames[providerNameString]
	if !ok {
		return fmt.Errorf("unknown Symbols provider name: '%s'", providerNameString)
	}
	*n = providerName
	return nil
}
