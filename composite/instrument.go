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

type SymbolCache cache.KeyValue[string, []finance.Instrument]

type cachedSymbolsProvider struct {
	provider finance.InstrumentProvider
	cache    SymbolCache
}

func NewCachedSymbolsProvider(provider finance.InstrumentProvider, cache SymbolCache) finance.InstrumentProvider {
	return &cachedSymbolsProvider{
		provider: provider,
		cache:    cache,
	}
}

func (p *cachedSymbolsProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("cached:")
	buffer.WriteString(p.provider.ProviderName())
	return buffer.String()
}

func (p *cachedSymbolsProvider) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	key := p.cacheKey(query)
	cachedSymbols, err := p.cache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedSymbols, err = p.provider.SearchInstruments(ctx, query)
		if err != nil {
			return nil, err
		}
		p.cache.Put(ctx, key, cachedSymbols)
	} else if err != nil {
		return nil, err
	}
	return cachedSymbols, nil
}

func (p *cachedSymbolsProvider) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	return instruments, nil
}

func (p *cachedSymbolsProvider) cacheKey(query string) string {
	return fmt.Sprintf("finance:symbols:%s", strings.ToUpper(query))
}

type mergeSymbolsProvider struct {
	queue *cooldownQueue[finance.InstrumentProvider]
}

func NewMergeSymbolsProvider(provider finance.InstrumentProvider, cooldown time.Duration, fallbacks ...finance.InstrumentProvider) finance.InstrumentProvider {
	return &mergeSymbolsProvider{queue: newCooldownQueue(provider, cooldown, fallbacks...)}
}

func (p *mergeSymbolsProvider) ProviderName() string {
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

func (p *mergeSymbolsProvider) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	availableProviders := p.queue.GetAvailableProviders()
	symbols := make([]finance.Instrument, 0)
	for _, availableProvider := range availableProviders {
		foundSymbols, err := availableProvider.SearchInstruments(ctx, query)
		if errors.Is(err, finance.ErrInstrumentSearchRestricted) {
			continue
		} else if err != nil {
			slog.Warn("marking Symbols provider as failed", slog.String("provider", availableProvider.ProviderName()), slog.Any("err", err))
			p.queue.MarkProviderFailed(availableProvider)
			continue
		}
		for _, foundSymbol := range foundSymbols {
			_ = foundSymbol
		}
	}
	return symbols, nil
}

func (p *mergeSymbolsProvider) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	return instruments, nil
}
