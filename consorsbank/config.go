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

import "crypto/tls"

type Config interface {
	GetAddress() (string, error)
	GetTLSConfig() (*tls.Config, error)
	GetSecret() (string, error)
}

type StaticConfig struct {
	Address   string
	TLSConfig *tls.Config
	Secret    string
}

func (c *StaticConfig) GetAddress() (string, error) {
	return c.Address, nil
}

func (c *StaticConfig) GetTLSConfig() (*tls.Config, error) {
	return c.TLSConfig, nil
}

func (c *StaticConfig) GetSecret() (string, error) {
	return c.Secret, nil
}
