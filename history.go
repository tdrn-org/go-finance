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

import "time"

type Interval int

const (
	IntervalNone   Interval = 0
	IntervalHourly Interval = 60
	IntervalDaily  Interval = 60 * 24
)

func FromToInterval(from, to time.Time) Interval {
	delta := to.Sub(from)
	switch {
	case delta <= 0:
		return IntervalNone
	case delta < 24*time.Hour:
		return IntervalHourly
	default:
		return IntervalDaily
	}
}
