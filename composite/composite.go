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
	entries map[string]cooldownQueueEntry[P]
	mutex   sync.RWMutex
}

func newCooldownQueue[P finance.APIProvider]() *cooldownQueue[P] {
	queue := &cooldownQueue[P]{
		entries: make(map[string]cooldownQueueEntry[P], 0),
	}
	return queue
}

func (q *cooldownQueue[P]) Add(provider P, cooldownTTL time.Duration) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.entries[provider.ProviderName()] = cooldownQueueEntry[P]{provider: provider, cooldownTTL: cooldownTTL}
}

func (q *cooldownQueue[P]) ForEach(f func(provider P, lastErr error)) {
	for _, entry := range q.providerEntries() {
		f(entry.provider, entry.lastErr)
	}
}

func (q *cooldownQueue[P]) providerEntries() []cooldownQueueEntry[P] {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	now := time.Now()
	entries := make([]cooldownQueueEntry[P], 0, len(q.entries))
	for providerName, providerEntry := range q.entries {
		if providerEntry.lastErr != nil && now.After(providerEntry.cooldownUntil) {
			providerEntry.lastErr = nil
			q.entries[providerName] = providerEntry
		}
		entries = append(entries, providerEntry)
	}
	return entries
}

func (q *cooldownQueue[P]) MarkProviderFailed(provider P, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	providerName := provider.ProviderName()
	providerEntry, ok := q.entries[providerName]
	if !ok {
		slog.Warn("unknown provider", slog.String("provider", providerName))
		return
	}
	slog.Warn(fmt.Sprintf("marking %T provider as failed", reflect.TypeFor[P]()), slog.String("provider", providerName), slog.Any("err", err))
	providerEntry.lastErr = err
	providerEntry.cooldownUntil = time.Now().Add(providerEntry.cooldownTTL)
	q.entries[providerName] = providerEntry
}
