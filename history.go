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
	"iter"
	"slices"
)

type History string

const (
	History24h History = "24h"
	History7D  History = "7d"
	History30D History = "30d"
)

type Bars []Bar

func (bars Bars) Sorted() iter.Seq2[int, Bar] {
	return bars.SortFunc(func(i, j int) int {
		return bars[i].Timestamp.Compare(bars[j].Timestamp)
	})
}

func (bars Bars) Reversed() iter.Seq2[int, Bar] {
	return bars.SortFunc(func(i, j int) int {
		return bars[j].Timestamp.Compare(bars[i].Timestamp)
	})
}

func (bars Bars) SortFunc(cmp func(int, int) int) iter.Seq2[int, Bar] {
	indices := make([]int, len(bars))
	for i := range indices {
		indices[i] = i
	}
	slices.SortFunc(indices, func(i, j int) int {
		return cmp(i, j)
	})
	return func(yield func(int, Bar) bool) {
		for _, originalIdx := range indices {
			if !yield(originalIdx, bars[originalIdx]) {
				return
			}
		}
	}
}
