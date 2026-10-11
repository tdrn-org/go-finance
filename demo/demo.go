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

// Package demo implements demo providers for testing purposes.
package demo

import (
	"log/slog"
)

// Name of the Demo Provider.
const Name string = "demo"

// API provides a Demo provider implementing
//   - [finance.FX]
//   - [finance.ListingProvider]
type API struct {
	enabled bool
	logger  *slog.Logger
}

// NewAPI creates a new Demo provider instance using the given [Config].
func NewAPI(config Config) (*API, error) {
	logger := slog.With(slog.String("provider", Name))
	api := &API{
		enabled: config.GetEnabled(),
		logger:  logger,
	}
	return api, nil
}

// See [finance.APIProvider.ProviderName]
func (api *API) ProviderName() string {
	return Name
}
