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
	"net/http"

	"github.com/tdrn-org/go-finance/twelvedata"
)

type TwelveDataConfig struct {
	APIKey  string                                         `toml:"api_key"`
	factory apiFactory[*twelvedata.API, twelvedata.Config] `toml:"-"`
}

func (c *TwelveDataConfig) GetAPIKey() (string, error) {
	return c.APIKey, nil
}

func (c *TwelveDataConfig) GetHttpClient() (*http.Client, error) {
	return nil, nil
}

func (c *TwelveDataConfig) NewAPI() (*twelvedata.API, error) {
	return c.factory.NewAPI(twelvedata.NewAPI, c)
}
