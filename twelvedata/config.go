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

package twelvedata

import "net/http"

// Config interface for Twelve Data provider.
type Config interface {
	// GetAPIKey gets the API Key used to access the Twelve Data API.
	GetAPIKey() (string, error)
	// GetHttpClient gets the [http.Client] used to access the Twelve Data API.
	// If nil is returned, the default [http.Client] is used.
	GetHttpClient() (*http.Client, error)
	// GetIncludeMICs gets the list of MICs to include in search results.
	// In case of an empty list, all MICs are included in search results.
	GetIncludeMICs() ([]string, error)
	// GetIncludeInstrumentTypes gets the list of instrument type to include
	// in search results. In case of an empty list, all instrument types are
	// included in search results.
	GetIncludeInstrumentTypes() ([]string, error)
}

// StaticConfig implements [Config] via static attributes.
type StaticConfig struct {
	// See [Config.GetAPIKey]
	APIKey string
	// See [Config.GetHttpClient]
	HttpClient *http.Client
	// See [Config.GetIncludeMICs]
	IncludeMICs []string
	// See [Config.GetIncludeInstrumentTypes]
	IncludeInstrumentTypes []string
}

// See [Config.GetAPIKey]
func (c *StaticConfig) GetAPIKey() (string, error) {
	return c.APIKey, nil
}

// See [Config.GetHttpClient]
func (c *StaticConfig) GetHttpClient() (*http.Client, error) {
	return c.HttpClient, nil
}

// See [Config.GetIncludeMICs]
func (c *StaticConfig) GetIncludeMICs() ([]string, error) {
	return c.IncludeMICs, nil
}

// See [Config.GetIncludeInstrumentTypes]
func (c *StaticConfig) GetIncludeInstrumentTypes() ([]string, error) {
	return c.IncludeInstrumentTypes, nil
}
