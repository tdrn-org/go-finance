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
	"github.com/tdrn-org/go-finance/demo"
	"github.com/tdrn-org/go-finance/frankfurter"
	"github.com/tdrn-org/go-finance/twelvedata"
)

type FXConfig struct {
	ProviderNames []FXProviderName `toml:"providers"`
	Cooldown      DurationSpec     `toml:"cooldown"`
	CacheTTL      DurationSpec     `toml:"cache_ttl"`
}

type FXProviderName string

const (
	FXProviderNameDemo         FXProviderName = FXProviderName(demo.Name)
	FXProviderNameAlphaVantage FXProviderName = FXProviderName(alphavantage.Name)
	FXProviderNameConsorsbank  FXProviderName = FXProviderName(consorsbank.Name)
	FXProviderNameFrankfurter  FXProviderName = FXProviderName(frankfurter.Name)
	FXProviderNameTwelveData   FXProviderName = FXProviderName(twelvedata.Name)
)

var knownFXProviderNames map[string]FXProviderName = map[string]FXProviderName{
	string(FXProviderNameDemo):         FXProviderNameDemo,
	string(FXProviderNameAlphaVantage): FXProviderNameAlphaVantage,
	string(FXProviderNameConsorsbank):  FXProviderNameConsorsbank,
	string(FXProviderNameFrankfurter):  FXProviderNameFrankfurter,
	string(FXProviderNameTwelveData):   FXProviderNameTwelveData,
}

func (n *FXProviderName) Value() string {
	for value, providerName := range knownFXProviderNames {
		if *n == providerName {
			return value
		}
	}
	slog.Warn("unexpected FX provider name", slog.Any("n", *n))
	return ""
}

func (n *FXProviderName) MarshalTOML() ([]byte, error) {
	return []byte(`"` + n.Value() + `"`), nil
}

func (n *FXProviderName) UnmarshalTOML(value any) error {
	providerNameString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected FX provider name type %v", value)
	}
	providerName, ok := knownFXProviderNames[providerNameString]
	if !ok {
		return fmt.Errorf("unknown FX provider name: '%s'", providerNameString)
	}
	*n = providerName
	return nil
}
