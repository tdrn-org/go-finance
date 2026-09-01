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
	"github.com/tdrn-org/go-finance/twelvedata"
)

type EquityConfig struct {
	ProviderName EquityProviderName  `toml:"provider"`
	CacheTTL     config.DurationSpec `toml:"cache_ttl"`
}

type EquityProviderName string

const (
	EquityProviderNameDemo         EquityProviderName = EquityProviderName(demo.Name)
	EquityProviderNameAlphaVantage EquityProviderName = EquityProviderName(alphavantage.Name)
	EquityProviderNameConsorsbank  EquityProviderName = EquityProviderName(consorsbank.Name)
	EquityProviderNameTwelveData   EquityProviderName = EquityProviderName(twelvedata.Name)
)

func (n EquityProviderName) String() string {
	return string(n)
}

var equityProviderNameUnmarshalMap map[string]EquityProviderName = map[string]EquityProviderName{
	string(EquityProviderNameDemo):         EquityProviderNameDemo,
	string(EquityProviderNameAlphaVantage): EquityProviderNameAlphaVantage,
	string(EquityProviderNameConsorsbank):  EquityProviderNameConsorsbank,
	string(EquityProviderNameTwelveData):   EquityProviderNameTwelveData,
}

func (n EquityProviderName) MarshalText() ([]byte, error) {
	return config.MarshalStringerEnum(n)
}

func (n *EquityProviderName) UnmarshalText(text []byte) error {
	providerName, err := config.UnmarshalEnum(equityProviderNameUnmarshalMap, text)
	if err != nil {
		return err
	}
	*n = providerName
	return nil
}
