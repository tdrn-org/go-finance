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
		providers := make([]finance.FX, 0, len(c.FX.ProviderNames))
		for _, providerName := range c.FX.ProviderNames {
			var provider finance.FX
			var err error
			switch providerName {
			case FXProviderNameDemo:
				provider, err = c.Demo.NewAPI()
			case FXProviderNameAlphaVantage:
				provider, err = c.AlphaVantage.NewAPI()
			case FXProviderNameConsorsbank:
				provider, err = c.Consorsbank.NewAPI()
			case FXProviderNameFrankfurter:
				provider, err = c.Frankfurter.NewAPI()
			case FXProviderNameTwelveData:
				provider, err = c.TwelveData.NewAPI()
			default:
				err = fmt.Errorf("unrecognized FX provider name %q", providerName)
			}
			if err != nil {
				return nil, err
			}
			providers = append(providers, provider)
		}
		if len(providers) == 0 {
			return nil, fmt.Errorf("at least one FX provider must be defined")
		}
		cache, err := c.Cache.NewExchangeRateCache(time.Duration(c.FX.CacheTTL))
		if err != nil {
			return nil, err
		}
		fallbackProvider := composite.NewFallbackFXProvider(time.Duration(c.FX.Cooldown), providers...)
		return composite.NewCachedFXProvider(fallbackProvider, cache), nil
	}, c)
	return c.fxFactory.api, c.fxFactory.err
}

func (c *Config) NewListingProvider() (finance.ListingProvider, error) {
	c.listingFactory.NewAPI(func(c *Config) (finance.ListingProvider, error) {
		providers := make([]finance.ListingProvider, 0, len(c.Listings.ProviderNames))
		for _, providerName := range c.Listings.ProviderNames {
			var provider finance.ListingProvider
			var err error
			switch providerName {
			case ListingProviderNameDemo:
				provider, err = c.Demo.NewAPI()
			case ListingProviderNameAlphaVantage:
				provider, err = c.AlphaVantage.NewAPI()
			case ListingProviderNameConsorsbank:
				provider, err = c.Consorsbank.NewAPI()
			case ListingProviderNameTwelveData:
				provider, err = c.TwelveData.NewAPI()
			default:
				err = fmt.Errorf("unrecognized Listing provider name %q", providerName)
			}
			if err != nil {
				return nil, err
			}
			providers = append(providers, provider)
		}
		if len(providers) == 0 {
			return nil, fmt.Errorf("at least one Listing provider must be defined")
		}
		listingCache, err := c.Cache.NewListingCache(time.Duration(c.Listings.ListingCacheTTL))
		if err != nil {
			return nil, err
		}
		quoteCache, err := c.Cache.NewQuoteCache(time.Duration(c.Listings.QuoteCacheTTL))
		if err != nil {
			return nil, err
		}
		cachedProviders := composite.NewCachedListingProviders(providers, listingCache, quoteCache)
		return composite.NewCombinedListingProvider(time.Duration(c.Listings.Cooldown), cachedProviders...), nil
	}, c)
	return c.listingFactory.api, c.listingFactory.err
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
