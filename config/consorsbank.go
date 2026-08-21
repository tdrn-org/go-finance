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
	"crypto/tls"
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank"
)

type ConsorsbankConfig struct {
	Address             string                                           `toml:"address"`
	SkipVerify          bool                                             `toml:"skip_verify"`
	CAFile              string                                           `toml:"ca_file"`
	Secret              string                                           `toml:"secret"`
	PreferredCurrency   finance.Currency                                 `toml:"preferred_currency"`
	PreferredExchanges  []string                                         `toml:"preferred_exchanges"`
	SubscriptionTimeout DurationSpec                                     `toml:"subscription_timeout"`
	factory             apiFactory[*consorsbank.API, consorsbank.Config] `toml:"-"`
}

func (c *ConsorsbankConfig) GetAddress() (string, error) {
	return c.Address, nil
}

func (c *ConsorsbankConfig) GetTLSConfig() (*tls.Config, error) {
	if c.SkipVerify {
		return consorsbank.TLSSkipVerify(), nil
	}
	return consorsbank.TLSCAFromFile(c.CAFile)
}

func (c *ConsorsbankConfig) GetSecret() (string, error) {
	return c.Secret, nil
}

func (c *ConsorsbankConfig) GetPreferredCurrency() (finance.Currency, error) {
	return c.PreferredCurrency, nil
}

func (c *ConsorsbankConfig) GetPreferredExchanges() ([]string, error) {
	return c.PreferredExchanges, nil
}

func (c *ConsorsbankConfig) GetSubscriptionTimeout() (time.Duration, error) {
	return time.Duration(c.SubscriptionTimeout), nil
}

func (c *ConsorsbankConfig) NewAPI() (*consorsbank.API, error) {
	return c.factory.NewAPI(consorsbank.NewAPI, c)
}
