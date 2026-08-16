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

package finance

import (
	"context"
	"errors"
	"time"
)

var (
	ErrQuoteNotAvailable error = errors.New("quote not available")
)

// Quote represents a single price data point for a financial instrument.
type Quote struct {
	Symbol          Symbol
	Timestamp       time.Time
	Open            float64
	High            float64
	Low             float64
	Close           float64
	Price           float64
	Volume          int64
	Currency        Currency
	Source          string
	SourceTimestamp time.Time
}

// Equity provides quote data for equities, ETFs, and similar instruments.
type Equity interface {
	APIProvider

	// QueryQuote returns the latest quote for a symbol.
	// Returns ErrQuoteNotAvailable if the provider cannot handle the given symbol
	// (e.g. because it requires a ticker but none was provided).
	QueryQuote(ctx context.Context, symbol Symbol) (*Quote, error)
}
