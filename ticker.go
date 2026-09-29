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

// IsTicker checks whether the given string represents a Ticker symbol.
// This only verfies whether this a syntactically correct Ticker symbol,
// but not whether this Ticker symbol really exists.
func IsTicker(s string) bool {
	sLen := len(s)
	if sLen < 1 || 5 < sLen {
		return false
	}
	for _, c := range s {
		isAlphanumeric := ('0' <= c && c <= '9') || ('A' <= c && c <= 'Z')
		if !isAlphanumeric {
			return false
		}
	}
	return true
}
