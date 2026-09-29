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
	"regexp"
	"strings"
)

var figiPattern regexp.Regexp = *regexp.MustCompile("^[BCDFGHJKLMNPQRSTVWXYZ]{2}G[0-9BCDFGHJKLMNPQRSTVWXYZ]{8}[0-9]$")

type figiValidator struct {
	sum    int
	double bool
}

func (v *figiValidator) Validate(s string) bool {
	v.sum = 0
	v.double = false
	if !figiPattern.MatchString(s) {
		return false
	}
	expected := int(s[11] - '0')
	for i := 10; i >= 0; i-- {
		c := s[i]
		if '0' <= c && c <= '9' {
			cValue := int(c - '0')
			v.shift(cValue)
		} else {
			cValue := int(c - 'A' + 10)
			quotient := cValue / 10
			remainder := cValue % 10
			v.shift(quotient)
			v.shift(remainder)
		}
	}
	actual := (10 - (v.sum % 10)) % 10
	return actual == expected
}

func (v *figiValidator) shift(i int) {
	v.double = !v.double
	summand := i
	if v.double {
		summand *= 2
		if summand > 9 {
			summand = (summand / 10) + (summand % 10)
		}
	}
	v.sum += summand
}

// IsFIGI checks whether the given string represents a FIGI.
// This only verfies whether this a syntactically correct FIGI,
// but not whether this FIGI really exists.
func IsFIGI(s string) bool {
	return (&figiValidator{}).Validate(strings.ToUpper(s))
}
