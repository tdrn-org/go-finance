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
	_ "embed"
	"fmt"
	"sync"
	"time"

	"github.com/tdrn-org/go-config-toml"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/composite"
)

type Config struct {
	FX             FXConfig                                     `toml:"fx"`
	Listings       ListingsConfig                               `toml:"listings"`
	Demo           DemoConfig                                   `toml:"demo"`
	AlphaVantage   AlphaVantageConfig                           `toml:"alphavantage"`
	Consorsbank    ConsorsbankConfig                            `toml:"consorsbank"`
	Frankfurter    FrankfurterConfig                            `toml:"frankfurter"`
	OpenFIGI       OpenFIGIConfig                               `toml:"openfigi"`
	TwelveData     TwelveDataConfig                             `toml:"twelvedata"`
	Cache          CacheConfig                                  `toml:"cache"`
	fxFactory      apiFactory[finance.FX, *Config]              `toml:"-"`
	listingFactory apiFactory[finance.ListingProvider, *Config] `toml:"-"`
}

//go:embed defaults.toml
var defaultsData []byte

func Default() (*Config, error) {
	return config.Defaults(&Config{}, defaultsData)
}

func Load(path string, strict bool) (*Config, error) {
	return config.Load(&Config{}, path, defaultsData, strict)
}

func (c *Config) NewFXProvider() (finance.FX, error) {
	c.fxFactory.NewAPI(func(c *Config) (finance.FX, error) {
		if len(c.FX.Providers) == 0 {
			return nil, fmt.Errorf("at least one FX provider must be defined")
		}
		fallbackProvider := composite.NewFallbackFXProvider()
		for _, providerConfig := range c.FX.Providers {
			provider, err := c.newUpstreamFXProvider(&providerConfig)
			if err != nil {
				return nil, err
			}
			cache, err := c.Cache.NewExchangeRateCache(time.Duration(providerConfig.CacheTTL))
			if err != nil {
				return nil, err
			}
			fallbackProvider.Add(composite.NewCachedFXProvider(provider, cache), time.Duration(providerConfig.CooldownTTL))
		}
		return fallbackProvider, nil
	}, c)
	return c.fxFactory.api, c.fxFactory.err
}

func (c *Config) newUpstreamFXProvider(providerConfig *FXProviderConfig) (finance.FX, error) {
	switch providerConfig.Name {
	case FXProviderNameDemo:
		return c.Demo.NewAPI()
	case FXProviderNameAlphaVantage:
		return c.AlphaVantage.NewAPI()
	case FXProviderNameConsorsbank:
		return c.Consorsbank.NewAPI()
	case FXProviderNameFrankfurter:
		return c.Frankfurter.NewAPI()
	case FXProviderNameTwelveData:
		return c.TwelveData.NewAPI()
	default:
		return nil, fmt.Errorf("unrecognized FX provider name %q", providerConfig.Name)
	}
}

func (c *Config) NewListingProvider() (finance.ListingProvider, error) {
	c.listingFactory.NewAPI(func(c *Config) (finance.ListingProvider, error) {
		if len(c.Listings.Providers) == 0 {
			return nil, fmt.Errorf("at least one Listing provider must be defined")
		}
		combinedProvider := composite.NewCombinedListingProvider()
		for _, providerConfig := range c.Listings.Providers {
			provider, err := c.newUpstreamListingProvider(&providerConfig)
			if err != nil {
				return nil, err
			}
			listingCache, err := c.Cache.NewListingCache(time.Duration(providerConfig.ListingCacheTTL))
			if err != nil {
				return nil, err
			}
			quoteCache, err := c.Cache.NewQuoteCache(time.Duration(providerConfig.QuoteCacheTTL))
			if err != nil {
				return nil, err
			}
			quoteHistoryCache, err := c.Cache.NewQuoteHistoryCache(time.Duration(providerConfig.QuoteHistoryCacheTTL))
			if err != nil {
				return nil, err
			}
			combinedProvider.Add(composite.NewCachedListingProvider(provider, listingCache, quoteCache, quoteHistoryCache), time.Duration(providerConfig.CooldownTTL))
		}
		return combinedProvider, nil
	}, c)
	return c.listingFactory.api, c.listingFactory.err
}

func (c *Config) newUpstreamListingProvider(providerConfig *ListingProviderConfig) (finance.ListingProvider, error) {
	switch providerConfig.Name {
	case ListingProviderNameDemo:
		return c.Demo.NewAPI()
	case ListingProviderNameAlphaVantage:
		return c.AlphaVantage.NewAPI()
	case ListingProviderNameConsorsbank:
		return c.Consorsbank.NewAPI()
	case ListingProviderNameTwelveData:
		return c.TwelveData.NewAPI()
	default:
		return nil, fmt.Errorf("unrecognized Listing provider name %q", providerConfig.Name)
	}
}

type apiFactory[A finance.APIProvider, C any] struct {
	new sync.Once
	api A
	err error
}

func (f *apiFactory[A, C]) NewAPI(newAPI func(config C) (A, error), config C) (A, error) {
	f.new.Do(func() {
		f.api, f.err = newAPI(config)
	})
	return f.api, f.err
}
