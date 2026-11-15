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

package consorsbank_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance/consorsbank"
)

func TestTradegateTradingDurationSince(t *testing.T) {
	since := time.Now().Add(-7 * 24 * time.Hour)
	duration := consorsbank.TradegateTradingDurationSince(since)
	require.Greaterf(t, duration, 0*time.Second, since.String())
	require.LessOrEqualf(t, duration, 73*time.Hour, since.String())
}

func TestTradegateTradingHours(t *testing.T) {
	location, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	years := []int{2026, 2027, 2028, 2029, 2030, 2031, 2032}
	closedDays := make([]time.Time, 0)
	for _, year := range years {
		days := []time.Time{
			time.Date(year, time.January, 1, 10, 00, 0, 0, location),   // New year's day
			time.Date(year, time.May, 1, 10, 00, 0, 0, location),       // Labour day
			time.Date(year, time.December, 24, 10, 00, 0, 0, location), // Christmas eve
			time.Date(year, time.December, 25, 10, 00, 0, 0, location), // Christmas day
			time.Date(year, time.December, 26, 10, 00, 0, 0, location), // 2nd Day of Christmas
			time.Date(year, time.December, 31, 10, 00, 0, 0, location), // New year's eve
		}
		closedDays = append(closedDays, days...)
	}
	for _, closedDay := range closedDays {
		open, _, _ := consorsbank.TradegateTradingHours(closedDay)
		require.Falsef(t, open, closedDay.String())
	}

	openDays := []time.Time{
		time.Date(2026, time.January, 2, 10, 00, 0, 0, location),
		time.Date(2026, time.April, 30, 10, 00, 0, 0, location),
		time.Date(2026, time.October, 2, 10, 00, 0, 0, location),
		time.Date(2026, time.December, 23, 10, 00, 0, 0, location),
		time.Date(2026, time.December, 28, 10, 00, 0, 0, location),
	}
	for _, openDay := range openDays {
		open, _, _ := consorsbank.TradegateTradingHours(openDay)
		require.Truef(t, open, openDay.String())
	}
}
