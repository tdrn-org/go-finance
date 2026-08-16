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

package alphavantage

import (
	"fmt"
	"strconv"
	"time"
)

func stringToTimestamp(s, name string) (time.Time, error) {
	timestamp, err := time.Parse(time.DateTime, s)
	if err != nil {
		timestamp, err = time.Parse(time.DateOnly, s)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse %s '%s' (cause: %w)", name, s, err)
	}
	return timestamp, nil
}

func stringToFloat64(s, name string) (float64, error) {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse %s '%s' (cause: %w)", name, s, err)
	}
	return value, nil
}

func stringToInt64(s, name string) (int64, error) {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse %s '%s' (cause: %w)", name, s, err)
	}
	return value, nil
}
