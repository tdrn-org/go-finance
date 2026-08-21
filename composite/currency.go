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

type ExchangeRateCache cache.KeyValue[string, *finance.ExchangeRate]

type cachedFXProvider struct {
	provider finance.FX
	cache    ExchangeRateCache
}

func NewCachedFXProvider(provider finance.FX, cache ExchangeRateCache) finance.FX {
	return &cachedFXProvider{
		provider: provider,
		cache:    cache,
	}
}

func (p *cachedFXProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("cached:")
	buffer.WriteString(p.provider.ProviderName())
	return buffer.String()
}

func (p *cachedFXProvider) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	key := p.cacheKey(base, quote)
	cachedExchangeRate, err := p.cache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedExchangeRate, err = p.provider.QueryExchangeRate(ctx, base, quote)
		if err != nil {
			return nil, err
		}
		p.cache.Put(ctx, key, cachedExchangeRate)
	} else if err != nil {
		return nil, err
	}
	return cachedExchangeRate, nil
}

func (p *cachedFXProvider) cacheKey(base, quote finance.Currency) string {
	return fmt.Sprintf("finance:exchangeRate:%s/%s", strings.ToUpper(string(base)), strings.ToUpper(string(quote)))
}

type fallbackFXProvider struct {
	queue *cooldownQueue[finance.FX]
}

func NewFallbackFXProvider(provider finance.FX, cooldown time.Duration, fallbacks ...finance.FX) finance.FX {
	return &fallbackFXProvider{
		queue: newCooldownQueue(provider, cooldown, fallbacks...),
	}
}

func (p *fallbackFXProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("fallback:")
	initialBufferLen := buffer.Len()
	p.queue.ForEach(func(provider finance.FX) {
		if buffer.Len() > initialBufferLen {
			buffer.WriteRune('|')
		}
		buffer.WriteString(provider.ProviderName())
	})
	return buffer.String()
}

func (p *fallbackFXProvider) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	availableProviders := p.queue.GetAvailableProviders()
	for _, availableProvider := range availableProviders {
		exchangeRate, err := availableProvider.QueryExchangeRate(ctx, base, quote)
		if errors.Is(err, finance.ErrRequestPending) {
			continue
		} else if err != nil {
			slog.Warn("marking FX provider as failed", slog.String("provider", availableProvider.ProviderName()), slog.Any("err", err))
			p.queue.MarkProviderFailed(availableProvider)
			continue
		}
		return exchangeRate, nil
	}
	return nil, finance.ErrExchangeRateNotAvailable
}
