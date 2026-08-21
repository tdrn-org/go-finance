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
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/composite"
)

type Config struct {
	FX            FXConfig                            `toml:"fx"`
	Equity        EquityConfig                        `toml:"equity"`
	AlphaVantage  AlphaVantageConfig                  `toml:"alphavantage"`
	Consorsbank   ConsorsbankConfig                   `toml:"consorsbank"`
	Frankfurter   FrankfurterConfig                   `toml:"frankfurter"`
	TwelveData    TwelveDataConfig                    `toml:"twelvedata"`
	Cache         CacheConfig                         `toml:"cache"`
	fxFactory     apiFactory[finance.FX, *Config]     `toml:"-"`
	equityFactory apiFactory[finance.Equity, *Config] `toml:"-"`
}

//go:embed defaults.toml
var defaultsData string

func Default() (*Config, error) {
	config := &Config{}
	meta, err := toml.Decode(defaultsData, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config defaults (cause: %w)", err)
	}
	for _, key := range meta.Undecoded() {
		slog.Warn("unexpected default configuration key", slog.Any("key", key))
	}
	return config, nil
}

func Load(path string, strict bool) (*Config, error) {
	logger := slog.With(slog.String("path", path))
	logger.Info("loading config")
	config, err := Default()
	if err != nil {
		return nil, err
	}
	meta, err := toml.DecodeFile(path, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config '%s' (cause: %w)", path, err)
	}
	strictViolation := false
	for _, key := range meta.Undecoded() {
		strictViolation = true
		logger.Warn("unexpected configuration key", slog.Any("key", key))
	}
	if strict && strictViolation {
		return nil, fmt.Errorf("config contains unexpected keys")
	}
	return config, nil
}

func (c *Config) NewFXProvider() (finance.FX, error) {
	c.fxFactory.NewAPI(func(c *Config) (finance.FX, error) {
		providers := make([]finance.FX, 0, len(c.FX.ProviderNames))
		for _, providerName := range c.FX.ProviderNames {
			var provider finance.FX
			var err error
			switch providerName {
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

func (c *Config) NewEquityProvider() (finance.Equity, error) {
	c.equityFactory.NewAPI(func(c *Config) (finance.Equity, error) {
		var provider finance.Equity
		var err error
		switch c.Equity.ProviderName {
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

type DurationSpec time.Duration

func (spec *DurationSpec) Value() string {
	return time.Duration(*spec).String()
}

func (spec *DurationSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *DurationSpec) UnmarshalTOML(value any) error {
	durationString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected duration type %v", value)
	}
	parsedDuration, err := time.ParseDuration(durationString)
	if err != nil {
		return fmt.Errorf("invalid duration: '%s' (cause: %w)", durationString, err)
	}
	*spec = DurationSpec(parsedDuration)
	return nil
}

type URLSpec struct {
	*url.URL
}

func (spec *URLSpec) Value() string {
	if spec.URL == nil {
		return ""
	}
	return spec.String()
}

func (spec *URLSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *URLSpec) UnmarshalTOML(value any) error {
	urlString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected URL type %v", value)
	}
	if urlString == "" {
		return nil
	}
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL: '%s' (cause: %w)", urlString, err)
	}
	spec.URL = parsedURL
	return nil
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
