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
	"github.com/tdrn-org/go-finance/twelvedata"
)

type EquityConfig struct {
	ProviderName EquityProviderName `toml:"provider"`
	CacheTTL     DurationSpec       `toml:"cache_ttl"`
}

type EquityProviderName string

const (
	EquityProviderNameAlphaVantage EquityProviderName = EquityProviderName(alphavantage.Name)
	EquityProviderNameConsorsbank  EquityProviderName = EquityProviderName(consorsbank.Name)
	EquityProviderNameTwelveData   EquityProviderName = EquityProviderName(twelvedata.Name)
)

var knownEquityProviderNames map[string]EquityProviderName = map[string]EquityProviderName{
	string(EquityProviderNameAlphaVantage): EquityProviderNameAlphaVantage,
	string(EquityProviderNameConsorsbank):  EquityProviderNameConsorsbank,
	string(EquityProviderNameTwelveData):   EquityProviderNameTwelveData,
}

func (n *EquityProviderName) Value() string {
	for value, providerName := range knownEquityProviderNames {
		if *n == providerName {
			return value
		}
	}
	slog.Warn("unexpected Equity provider name", slog.Any("n", *n))
	return ""
}

func (n *EquityProviderName) MarshalTOML() ([]byte, error) {
	return []byte(`"` + n.Value() + `"`), nil
}

func (n *EquityProviderName) UnmarshalTOML(value any) error {
	providerNameString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected Equity provider name type %v", value)
	}
	providerName, ok := knownEquityProviderNames[providerNameString]
	if !ok {
		return fmt.Errorf("unknown Equity provider name: '%s'", providerNameString)
	}
	*n = providerName
	return nil
}
