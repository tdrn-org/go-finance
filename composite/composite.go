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

package composite

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/tdrn-org/go-finance"
)

type cooldownQueueEntry[P finance.APIProvider] struct {
	provider      P
	cooldownTTL   time.Duration
	lastErr       error
	cooldownUntil time.Time
}

type cooldownQueue[P finance.APIProvider] struct {
	entries []cooldownQueueEntry[P]
	mutex   sync.RWMutex
}

func newCooldownQueue[P finance.APIProvider]() *cooldownQueue[P] {
	queue := &cooldownQueue[P]{
		entries: make([]cooldownQueueEntry[P], 0),
	}
	return queue
}

func (q *cooldownQueue[P]) Add(provider P, cooldownTTL time.Duration) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.entries = append(q.entries, cooldownQueueEntry[P]{provider: provider, cooldownTTL: cooldownTTL})
}

func (q *cooldownQueue[P]) ForEach(f func(provider P, lastErr error)) {
	for _, entry := range q.providerEntries() {
		f(entry.provider, entry.lastErr)
	}
}

func (q *cooldownQueue[P]) providerEntries() []cooldownQueueEntry[P] {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	now := time.Now()
	entries := make([]cooldownQueueEntry[P], 0, len(q.entries))
	for i, entry := range q.entries {
		if entry.lastErr != nil && now.After(entry.cooldownUntil) {
			entry.lastErr = nil
			q.entries[i] = entry
		}
		entries = append(entries, entry)
	}
	return entries
}

func (q *cooldownQueue[P]) MarkProviderFailed(provider P, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	providerName := provider.ProviderName()
	for i, entry := range q.entries {
		if entry.provider.ProviderName() == providerName {
			slog.Warn(fmt.Sprintf("marking %T provider as failed", reflect.TypeFor[P]()), slog.String("provider", providerName), slog.Any("err", err))
			entry.lastErr = err
			entry.cooldownUntil = time.Now().Add(entry.cooldownTTL)
			q.entries[i] = entry
			break
		}
	}
}
