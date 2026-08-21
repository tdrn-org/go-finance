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
	// ErrQuoteNotAvailable indicates a provider is not able to
	// provide a quote for the given [Symbol]. This is a permanent
	// error due to missing Symbol ids required by the provider.
	ErrQuoteNotAvailable error = errors.New("quote not available")
)

// Quote represents a single price data point for a financial instrument.
type Quote struct {
	// Symbol identifies the financial instrument by various ids.
	Symbol Symbol `json:"symbol"`
	// Timestamp gives the point in time this quote was current
	// according to the sourcing provider.
	Timestamp time.Time `json:"timestamp"`
	// Open gives the open price at the trading day identified by Timestamp.
	Open float64 `json:"open"`
	// High gives the high price at the trading day identified by Timestamp.
	High float64 `json:"high"`
	// Low gives the low price at the trading day identified by Timestamp.
	Low float64 `json:"low"`
	// Close gives the close price of the day before the trading day identified by Timestamp.
	Close float64 `json:"close"`
	// Price gives the current price at the trading day identified by Timestamp.
	Price float64 `json:"price"`
	// Volume gives the order volume at the trading day identified by Timestamp.
	Volume int64 `json:"volume"`
	// Currency gives the currency the given values.
	Currency Currency `json:"currency"`
	// Sources defines the provider this quote has been
	// queried from.
	Source string `json:"source"`
	// SourceTimestamp defines the time this quote has
	// been queried.
	SourceTimestamp time.Time `json:"source_timestamp"`
}

// Equity provides quote data for equities, ETFs, and similar instruments.
type Equity interface {
	APIProvider

	// QueryQuote returns the latest quote for a symbol.
	// Returns ErrQuoteNotAvailable if the provider cannot handle the given symbol
	// (e.g. because it requires a ticker but none was provided).
	QueryQuote(ctx context.Context, symbol Symbol) (*Quote, error)
}
