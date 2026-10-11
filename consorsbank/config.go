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

package consorsbank

import (
	"crypto/tls"
	"time"

	"github.com/tdrn-org/go-finance"
)

// DefaultAddress defines the default address of the Consorsbank TAPI server.
const DefaultAddress string = "localhost:40443"

// Config interface for Consorsbank TAPI Provider.
type Config interface {
	// GetAddress gets the address (host:port) the TAPI service is listening on.
	// [DefaultAddress] is used if empty.
	GetAddress() (string, error)
	// GetTLSConfig gets the TLS config to use to connect to the TAPI service.
	// Default [tls.Config] is used if nil.
	GetTLSConfig() (*tls.Config, error)
	// GetSecret gets the secret to use to connect to the TAPI service.
	GetSecret() (string, error)
	// GetIncludeExchanges get the TAPI exchange IDs to include during a search.
	// If the array is empty, all exchange IDs are included.
	GetIncludeExchanges() ([]string, error)
	// GetPreferredCurrency gets the preferred currency for quote queries. If
	// empty the listings default currency is used.
	GetPreferredCurrency() (finance.Currency, error)
	// GetSubscriptionTimeout gets the duration after which a no longer
	// accessed subscription is closed. If 0 is returned subscriptions
	// will not closed at all.
	GetSubscriptionTimeout() (time.Duration, error)
}

// StaticConfig implements [Config] via static attributes.
type StaticConfig struct {
	// See [Config.GetAddress]
	Address string
	// See [Config.GetTLSConfig]
	TLSConfig *tls.Config
	// See [Config.GetSecret]
	Secret string
	// See [Config.GetIncludeExchanges]
	IncludeExchanges []string
	// See [Config.GetPreferredCurrency]
	PreferredCurrency finance.Currency
	// See [Config.GetSubscriptionTimeout]
	SubscriptionTimeout time.Duration
}

// See [Config.GetAddress]
func (c *StaticConfig) GetAddress() (string, error) {
	return c.Address, nil
}

// See [Config.GetTLSConfig]
func (c *StaticConfig) GetTLSConfig() (*tls.Config, error) {
	return c.TLSConfig, nil
}

// See [Config.GetSecret]
func (c *StaticConfig) GetSecret() (string, error) {
	return c.Secret, nil
}

// See [Config.GetIncludeExchanges]
func (c *StaticConfig) GetIncludeExchanges() ([]string, error) {
	return c.IncludeExchanges, nil
}

// See [Config.GetPreferredCurrency]
func (c *StaticConfig) GetPreferredCurrency() (finance.Currency, error) {
	return c.PreferredCurrency, nil
}

// See [Config.GetSubscriptionTimeout]
func (c *StaticConfig) GetSubscriptionTimeout() (time.Duration, error) {
	return c.SubscriptionTimeout, nil
}
