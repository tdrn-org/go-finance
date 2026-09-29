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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tdrn-org/go-cache"
	"github.com/tdrn-org/go-finance"
)

type InstrumentCache cache.KeyValue[string, []finance.Instrument]

type cachedInstrumentProvider struct {
	provider finance.InstrumentProvider
	cache    InstrumentCache
}

func NewCachedInstrumentProvider(provider finance.InstrumentProvider, cache InstrumentCache) finance.InstrumentProvider {
	return &cachedInstrumentProvider{
		provider: provider,
		cache:    cache,
	}
}

func (p *cachedInstrumentProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("cached:")
	buffer.WriteString(p.provider.ProviderName())
	return buffer.String()
}

func (p *cachedInstrumentProvider) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	key := p.cacheKey(query)
	cachedInstruments, err := p.cache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedInstruments, err = p.provider.SearchInstruments(ctx, query)
		if err != nil {
			return nil, err
		}
		p.cache.Put(ctx, key, cachedInstruments)
	} else if err != nil {
		return nil, err
	}
	return cachedInstruments, nil
}

func (p *cachedInstrumentProvider) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	return instruments, nil
}

func (p *cachedInstrumentProvider) cacheKey(query string) string {
	return fmt.Sprintf("finance:symbols:%s", strings.ToUpper(query))
}

type mergeInstrumentProvider struct {
	queue *cooldownQueue[finance.InstrumentProvider]
}

func NewMergeInstrumentProvider(provider finance.InstrumentProvider, cooldown time.Duration, fallbacks ...finance.InstrumentProvider) finance.InstrumentProvider {
	return &mergeInstrumentProvider{queue: newCooldownQueue(provider, cooldown, fallbacks...)}
}

func (p *mergeInstrumentProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("merge:")
	initialBufferLen := buffer.Len()
	p.queue.ForEach(func(provider finance.InstrumentProvider) {
		if buffer.Len() > initialBufferLen {
			buffer.WriteRune('|')
		}
		buffer.WriteString(provider.ProviderName())
	})
	return buffer.String()
}

func (p *mergeInstrumentProvider) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	availableProviders := p.queue.GetAvailableProviders()
	instruments := make([]finance.Instrument, 0)
	for _, availableProvider := range availableProviders {
		foundInstruments, err := availableProvider.SearchInstruments(ctx, query)
		if errors.Is(err, finance.ErrInstrumentSearchRestricted) {
			continue
		} else if err != nil {
			slog.Warn("marking Symbols provider as failed", slog.String("provider", availableProvider.ProviderName()), slog.Any("err", err))
			p.queue.MarkProviderFailed(availableProvider)
			continue
		}
		for _, foundInstrument := range foundInstruments {
			//TODO: Merge
			_ = foundInstrument
		}
	}
	return instruments, nil
}

func (p *mergeInstrumentProvider) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	return instruments, nil
}
