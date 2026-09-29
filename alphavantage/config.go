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

package alphavantage

import (
	"net/http"
	"net/url"
)

const defaultBaseURLString string = "https://www.alphavantage.co/query/"

// DefaultBaseURL defines the default base URL of the Alpha Vantage REST API.
var DefaultBaseURL *url.URL = func() *url.URL {
	defaultBaseURL, err := url.Parse(defaultBaseURLString)
	if err != nil {
		panic(err)
	}
	return defaultBaseURL
}()

// Config interface for Alpha Vantage Provider.
type Config interface {
	// GetBaseURL gets the base URL to use for accessing the Alpha Vantage REST API.
	GetBaseURL() (*url.URL, error)
	// GetAPIKey gets the API key to use to access the Alpha Vantage REST API.
	GetAPIKey() (string, error)
	// GetHttpClient gets the [http.Client] to use to access the Alpha Vantage REST API.
	// If nil is returned, the default [http.Client] is used.
	GetHttpClient() (*http.Client, error)
}

// StaticConfig implements [Config] via static attributes.
type StaticConfig struct {
	BaseURL    *url.URL
	APIKey     string
	HttpClient *http.Client
}

// See [Config.GetBaseURL]
func (c *StaticConfig) GetBaseURL() (*url.URL, error) {
	return c.BaseURL, nil
}

// See [Config.GetAPIKey]
func (c *StaticConfig) GetAPIKey() (string, error) {
	return c.APIKey, nil
}

// See [Config.GetHttpClient]
func (c *StaticConfig) GetHttpClient() (*http.Client, error) {
	return c.HttpClient, nil
}
