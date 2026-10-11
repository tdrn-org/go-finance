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

package demo

// Config interface for Demo provider.
type Config interface {
	// GetEnabled determines whether the Demo provider is enabled or not.
	GetEnabled() bool
}

// StaticConfig implements [Config] via static attributes.
type StaticConfig struct {
	Enabled bool
}

// See [Config.GetEnabled]
func (c *StaticConfig) GetEnabled() bool {
	return c.Enabled
}
