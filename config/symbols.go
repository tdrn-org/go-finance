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
	"github.com/tdrn-org/go-config-toml"
	"github.com/tdrn-org/go-finance/alphavantage"
	"github.com/tdrn-org/go-finance/consorsbank"
	"github.com/tdrn-org/go-finance/demo"
	"github.com/tdrn-org/go-finance/openfigi"
	"github.com/tdrn-org/go-finance/twelvedata"
)

type SymbolsConfig struct {
	ProviderNames []SymbolsProviderName `toml:"providers"`
	Cooldown      config.DurationSpec   `toml:"cooldown"`
	CacheTTL      config.DurationSpec   `toml:"cache_ttl"`
}

type SymbolsProviderName string

const (
	SymbolsProviderNameDemo         SymbolsProviderName = SymbolsProviderName(demo.Name)
	SymbolsProviderNameAlphaVantage SymbolsProviderName = SymbolsProviderName(alphavantage.Name)
	SymbolsProviderNameConsorsbank  SymbolsProviderName = SymbolsProviderName(consorsbank.Name)
	SymbolsProviderNameOpenFIGI     SymbolsProviderName = SymbolsProviderName(openfigi.Name)
	SymbolsProviderNameTwelveData   SymbolsProviderName = SymbolsProviderName(twelvedata.Name)
)

func (n SymbolsProviderName) String() string {
	return string(n)
}

var symbolsProviderNamesUnmarshalMap map[string]SymbolsProviderName = map[string]SymbolsProviderName{
	string(SymbolsProviderNameDemo):         SymbolsProviderNameDemo,
	string(SymbolsProviderNameAlphaVantage): SymbolsProviderNameAlphaVantage,
	string(SymbolsProviderNameConsorsbank):  SymbolsProviderNameConsorsbank,
	string(SymbolsProviderNameOpenFIGI):     SymbolsProviderNameOpenFIGI,
	string(SymbolsProviderNameTwelveData):   SymbolsProviderNameTwelveData,
}

func (n SymbolsProviderName) MarshalText() ([]byte, error) {
	return config.MarshalStringerEnum(n)
}

func (n *SymbolsProviderName) UnmarshalText(text []byte) error {
	providerName, err := config.UnmarshalEnum(symbolsProviderNamesUnmarshalMap, text)
	if err != nil {
		return err
	}
	*n = providerName
	return nil
}
