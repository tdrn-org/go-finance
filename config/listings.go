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

type ListingsConfig struct {
	Providers []ListingProviderConfig `toml:"provider"`
}

type ListingProviderConfig struct {
	Name                 ListingProviderName `toml:"name"`
	CooldownTTL          config.DurationSpec `toml:"cooldown_ttl"`
	ListingCacheTTL      config.DurationSpec `toml:"listing_cache_ttl"`
	QuoteCacheTTL        config.DurationSpec `toml:"quote_cache_ttl"`
	QuoteHistoryCacheTTL config.DurationSpec `toml:"quote_history_cache_ttl"`
}

type ListingProviderName string

const (
	ListingProviderNameDemo         ListingProviderName = ListingProviderName(demo.Name)
	ListingProviderNameAlphaVantage ListingProviderName = ListingProviderName(alphavantage.Name)
	ListingProviderNameConsorsbank  ListingProviderName = ListingProviderName(consorsbank.Name)
	ListingProviderNameTwelveData   ListingProviderName = ListingProviderName(twelvedata.Name)
)

func (n ListingProviderName) String() string {
	return string(n)
}

var listingsProviderNamesUnmarshalMap map[string]ListingProviderName = map[string]ListingProviderName{
	string(ListingProviderNameDemo):         ListingProviderNameDemo,
	string(ListingProviderNameAlphaVantage): ListingProviderNameAlphaVantage,
	string(ListingProviderNameConsorsbank):  ListingProviderNameConsorsbank,
	string(ListingProviderNameTwelveData):   ListingProviderNameTwelveData,
}

func (n ListingProviderName) MarshalText() ([]byte, error) {
	return config.MarshalStringerEnum(n)
}

func (n *ListingProviderName) UnmarshalText(text []byte) error {
	providerName, err := config.UnmarshalEnum(listingsProviderNamesUnmarshalMap, text)
	if err != nil {
		return err
	}
	*n = providerName
	return nil
}
