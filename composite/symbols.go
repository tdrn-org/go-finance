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

type SymbolCache cache.KeyValue[string, finance.Symbols]

type cachedSymbolsProvider struct {
	provider finance.SymbolResolver
	cache    SymbolCache
}

func NewCachedSymbolsProvider(provider finance.SymbolResolver, cache SymbolCache) finance.SymbolResolver {
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

func (p *cachedSymbolsProvider) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	key := fmt.Sprintf("finance:symbols:%s", query)
	cachedSymbols, err := p.cache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedSymbols, err = p.provider.SearchSymbol(ctx, query)
		if err != nil {
			return nil, err
		}
		p.cache.Put(ctx, key, cachedSymbols)
	} else if err != nil {
		return nil, err
	}
	return cachedSymbols, nil
}

type mergeSymbolsProvider struct {
	queue *cooldownQueue[finance.SymbolResolver]
}

func NewMergeSymbolsProvider(provider finance.SymbolResolver, cooldown time.Duration, fallbacks ...finance.SymbolResolver) finance.SymbolResolver {
	return &mergeSymbolsProvider{queue: newCooldownQueue(provider, cooldown, fallbacks...)}
}

func (p *mergeSymbolsProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("merge:")
	initialBufferLen := buffer.Len()
	p.queue.ForEach(func(provider finance.SymbolResolver) {
		if buffer.Len() > initialBufferLen {
			buffer.WriteRune('|')
		}
		buffer.WriteString(provider.ProviderName())
	})
	return buffer.String()
}

func (p *mergeSymbolsProvider) SearchSymbol(ctx context.Context, query string) (finance.Symbols, error) {
	availableProviders := p.queue.GetAvailableProviders()
	symbols := make(finance.Symbols, 0)
	for _, availableProvider := range availableProviders {
		foundSymbols, err := availableProvider.SearchSymbol(ctx, query)
		if err != nil {
			slog.Warn("marking Symbols provider as failed", slog.String("provider", availableProvider.ProviderName()), slog.Any("err", err))
			p.queue.MarkProviderFailed(availableProvider)
			continue
		}
		for _, foundSymbol := range foundSymbols {
			symbols = p.mergeSymbol(symbols, &foundSymbol)
		}
	}
	if len(symbols) == 0 {
		return nil, finance.ErrNoExchangeRate
	}
	return symbols, nil
}

func (p *mergeSymbolsProvider) mergeSymbol(symbols finance.Symbols, foundSymbol *finance.Symbol) finance.Symbols {
	if foundSymbol.IsEmpty() {
		return symbols
	}
	for i, symbol := range symbols {
		switch symbol.Match(foundSymbol) {
		case finance.SymbolMatchEqual:
			// Symbol is already in result; simply return
			return symbols
		case finance.SymbolMatchSoft:
			// Symbol is only partly in result; merge and return
			symbol.Merge(*foundSymbol)
			symbols[i] = symbol
			return symbols
		default:
			// Symbol may not be in list; continue search
		}
	}
	// Symbol is not yet in list; add it
	return append(symbols, *foundSymbol)
}
