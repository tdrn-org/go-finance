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
	"sync"
	"time"
)

type cooldownQueueEntry[P comparable] struct {
	provider      P
	cooldownUntil time.Time
}

type cooldownQueue[P comparable] struct {
	cooldown time.Duration
	entries  []*cooldownQueueEntry[P]
	mutex    sync.RWMutex
}

func newCooldownQueue[P comparable](provider P, cooldown time.Duration, fallbacks ...P) *cooldownQueue[P] {
	queue := &cooldownQueue[P]{
		cooldown: cooldown,
		entries:  make([]*cooldownQueueEntry[P], 0, 1+len(fallbacks)),
	}
	queue.entries = append(queue.entries, &cooldownQueueEntry[P]{provider: provider})
	for _, fallback := range fallbacks {
		queue.entries = append(queue.entries, &cooldownQueueEntry[P]{provider: fallback})
	}
	return queue
}

func (q *cooldownQueue[P]) ForEach(f func(provider P)) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	for _, entry := range q.entries {
		f(entry.provider)
	}
}

func (q *cooldownQueue[P]) GetAvailableProviders() []P {
	providers := make([]P, 0, len(q.entries))

	q.mutex.RLock()
	defer q.mutex.RUnlock()

	now := time.Now()
	for _, entry := range q.entries {
		if now.After(entry.cooldownUntil) {
			providers = append(providers, entry.provider)
		}
	}
	return providers
}

func (q *cooldownQueue[P]) MarkProviderFailed(provider P) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for _, entry := range q.entries {
		if entry.provider == provider {
			entry.cooldownUntil = time.Now().Add(q.cooldown)
			break
		}
	}
}
