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

import "time"

func TradegateTradingDurationSince(t time.Time) time.Duration {
	start := t.In(tradegateLocation)
	end := time.Now().In(tradegateLocation)
	if !start.Before(end) {
		return 0
	}
	var total time.Duration
	currentDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, tradegateLocation)
	for currentDay.Before(end) {
		open, marketOpen, marketClose := TradegateTradingHours(currentDay)
		if open {
			effectiveStart := marketOpen
			if start.After(effectiveStart) {
				effectiveStart = start
			}
			effectiveEnd := marketClose
			if end.Before(effectiveEnd) {
				effectiveEnd = end
			}
			if effectiveEnd.After(effectiveStart) {
				total += effectiveEnd.Sub(effectiveStart)
			}
		}
		currentDay = currentDay.AddDate(0, 0, 1)
	}
	return total
}

func TradegateTradingHours(t time.Time) (bool, time.Time, time.Time) {
	trgTime := t.In(tradegateLocation)
	weekday := trgTime.Weekday()

	// No trading on weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return false, time.Time{}, time.Time{}
	}

	year, month, day := trgTime.Date()

	// No trading on new year's day, labour day, christmas and new year's eve.
	if (month == time.January && day == 1) ||
		(month == time.May && day == 1) ||
		(month == time.December && (day == 24 || day == 25 || day == 26 || day == 31)) {
		return false, time.Time{}, time.Time{}
	}

	at := func(hour, minute int) time.Time {
		return time.Date(year, month, day, hour, minute, 0, 0, tradegateLocation)
	}

	easterSunday := easterSunday(year, tradegateLocation)
	today := at(0, 0)

	goodFriday := easterSunday.AddDate(0, 0, -2)
	easterMonday := easterSunday.AddDate(0, 0, 1)

	// No trading on easter
	if today.Equal(goodFriday) || today.Equal(easterMonday) {
		return false, time.Time{}, time.Time{}
	}

	ascensionDay := easterSunday.AddDate(0, 0, 39)
	whitMonday := easterSunday.AddDate(0, 0, 50)

	// Reduced hours on ascension day, whit mondy and unity day
	if today.Equal(ascensionDay) || today.Equal(whitMonday) || (month == time.October && day == 3) {
		return true, at(7, 30), at(20, 0)
	}

	// Reduced hours on last trading day of the year
	lastTradingDay := lastTradingDay(year, tradegateLocation)
	if today.Equal(lastTradingDay) {
		return true, at(7, 30), at(14, 0)
	}

	// Default trading hours
	return true, at(7, 30), at(22, 0)
}

func easterSunday(year int, location *time.Location) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
}

func lastTradingDay(year int, location *time.Location) time.Time {
	dec30 := time.Date(year, time.December, 30, 0, 0, 0, 0, location)
	switch dec30.Weekday() {
	case time.Saturday:
		return dec30.AddDate(0, 0, -1) // Friday 29.12.
	case time.Sunday:
		return dec30.AddDate(0, 0, -2) // Friday 28.12.
	default:
		return dec30
	}
}

var tradegateLocation *time.Location

func init() {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	tradegateLocation = location
}
