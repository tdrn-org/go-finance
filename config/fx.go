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
	"github.com/tdrn-org/go-finance/frankfurter"
	"github.com/tdrn-org/go-finance/twelvedata"
)

type FXConfig struct {
	ProviderNames []FXProviderName    `toml:"providers"`
	Cooldown      config.DurationSpec `toml:"cooldown"`
	CacheTTL      config.DurationSpec `toml:"cache_ttl"`
}

type FXProviderName string

const (
	FXProviderNameDemo         FXProviderName = FXProviderName(demo.Name)
	FXProviderNameAlphaVantage FXProviderName = FXProviderName(alphavantage.Name)
	FXProviderNameConsorsbank  FXProviderName = FXProviderName(consorsbank.Name)
	FXProviderNameFrankfurter  FXProviderName = FXProviderName(frankfurter.Name)
	FXProviderNameTwelveData   FXProviderName = FXProviderName(twelvedata.Name)
)

func (n FXProviderName) String() string {
	return string(n)
}

var fxProviderNameUnmarshalMap map[string]FXProviderName = map[string]FXProviderName{
	string(FXProviderNameDemo):         FXProviderNameDemo,
	string(FXProviderNameAlphaVantage): FXProviderNameAlphaVantage,
	string(FXProviderNameConsorsbank):  FXProviderNameConsorsbank,
	string(FXProviderNameFrankfurter):  FXProviderNameFrankfurter,
	string(FXProviderNameTwelveData):   FXProviderNameTwelveData,
}

func (n *FXProviderName) MarshalText() ([]byte, error) {
	return config.MarshalStringerEnum(n)
}

func (n *FXProviderName) UnmarshalText(text []byte) error {
	providerName, err := config.UnmarshalEnum(fxProviderNameUnmarshalMap, text)
	if err != nil {
		return err
	}
	*n = providerName
	return nil
}
