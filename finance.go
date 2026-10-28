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

// Package finance provides financial services backed up by different
// providers in a pluggable setup.
package finance

import (
	"context"
	"errors"
)

var (
	// ErrRateLimitReached indicates a provider will only respond
	// to further queries after a cool-down period.
	ErrRateLimitReached error = errors.New("rate limit reached")
	// ErrRequestPending indicates a request will be served
	// asynchronously and future calls may return the requested
	// data.
	ErrRequestPending error = errors.New("request pending")
)

// IsTransientErr determines whether the given error is a
// transient error, allowing a retry shortly after. A nil
// error is also considered transient.
func IsTransientErr(err error) bool {
	return err == nil || err == ErrRequestPending || err == ErrListingSearchRestricted
}

// APIProvider is the interface common to all providers.
type APIProvider interface {
	// ProviderName gets the name of this provider.
	ProviderName() string

	// Shutdown gracefully shuts down any running
	// background requests started by this provider.
	Shutdown(ctx context.Context) error

	// Close releases all resources allocated by this
	// provider.
	Close() error
}
