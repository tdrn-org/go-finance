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
	FX             FXConfig                                        `toml:"fx"`
	Symbols        SymbolsConfig                                   `toml:"symbols"`
	Equity         EquityConfig                                    `toml:"equity"`
	Demo           DemoConfig                                      `toml:"demo"`
	AlphaVantage   AlphaVantageConfig                              `toml:"alphavantage"`
	Consorsbank    ConsorsbankConfig                               `toml:"consorsbank"`
	Frankfurter    FrankfurterConfig                               `toml:"frankfurter"`
	OpenFIGI       OpenFIGIConfig                                  `toml:"openfigi"`
	TwelveData     TwelveDataConfig                                `toml:"twelvedata"`
	Cache          CacheConfig                                     `toml:"cache"`
	fxFactory      apiFactory[finance.FX, *Config]                 `toml:"-"`
	symbolsFactory apiFactory[finance.InstrumentProvider, *Config] `toml:"-"`
	equityFactory  apiFactory[finance.QuoteProvider, *Config]      `toml:"-"`
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
				err = fmt.Errorf("unrecognized FX provider name '%s'", providerName)
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
		return composite.NewCachedFXProvider(composite.NewFallbackFXProvider(providers[0], time.Duration(c.FX.Cooldown), providers[1:]...), cache), nil
	}, c)
	return c.fxFactory.api, c.fxFactory.err
}

func (c *Config) NewInstrumentProvider() (finance.InstrumentProvider, error) {
	c.symbolsFactory.NewAPI(func(c *Config) (finance.InstrumentProvider, error) {
		providers := make([]finance.InstrumentProvider, 0, len(c.Symbols.ProviderNames))
		for _, providerName := range c.Symbols.ProviderNames {
			var provider finance.InstrumentProvider
			var err error
			switch providerName {
			case SymbolsProviderNameDemo:
				provider, err = c.Demo.NewAPI()
			case SymbolsProviderNameAlphaVantage:
				provider, err = c.AlphaVantage.NewAPI()
			case SymbolsProviderNameConsorsbank:
				provider, err = c.Consorsbank.NewAPI()
			case SymbolsProviderNameOpenFIGI:
				provider, err = c.OpenFIGI.NewAPI()
			case SymbolsProviderNameTwelveData:
				provider, err = c.TwelveData.NewAPI()
			default:
				err = fmt.Errorf("unrecognized Symbols provider name '%s'", providerName)
			}
			if err != nil {
				return nil, err
			}
			providers = append(providers, provider)
		}
		if len(providers) == 0 {
			return nil, fmt.Errorf("at least one Symbols provider must be defined")
		}
		cache, err := c.Cache.NewInstrumentCache(time.Duration(c.Symbols.CacheTTL))
		if err != nil {
			return nil, err
		}
		return composite.NewCachedInstrumentProvider(composite.NewMergeInstrumentProvider(providers[0], time.Duration(c.FX.Cooldown), providers[1:]...), cache), nil
	}, c)
	return c.symbolsFactory.api, c.symbolsFactory.err
}

func (c *Config) NewQuoteProvider() (finance.QuoteProvider, error) {
	c.equityFactory.NewAPI(func(c *Config) (finance.QuoteProvider, error) {
		var provider finance.QuoteProvider
		var err error
		switch c.Equity.ProviderName {
		case EquityProviderNameDemo:
			provider, err = c.Demo.NewAPI()
		case EquityProviderNameAlphaVantage:
			provider, err = c.AlphaVantage.NewAPI()
		case EquityProviderNameConsorsbank:
			provider, err = c.Consorsbank.NewAPI()
		case EquityProviderNameTwelveData:
			provider, err = c.TwelveData.NewAPI()
		default:
			err = fmt.Errorf("unrecognized Equity provider name '%s'", c.Equity.ProviderName)
		}
		if err != nil {
			return nil, err
		}
		cache, err := c.Cache.NewQuoteCache(time.Duration(c.Equity.CacheTTL))
		if err != nil {
			return nil, err
		}
		return composite.NewCachedEquityProvider(provider, cache), nil
	}, c)
	return c.equityFactory.api, c.equityFactory.err
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
