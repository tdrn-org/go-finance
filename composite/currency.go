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
	"slices"
	"strings"
	"time"

	"github.com/tdrn-org/go-cache"
	"github.com/tdrn-org/go-finance"
)

type ExchangeRateCache cache.KeyValue[string, *finance.ExchangeRate]

type cachedFXProvider struct {
	provider finance.FX
	cache    ExchangeRateCache
	logger   *slog.Logger
}

func NewCachedFXProvider(provider finance.FX, cache ExchangeRateCache) finance.FX {
	return &cachedFXProvider{
		provider: provider,
		cache:    cache,
		logger:   slog.With(slog.String("provider", provider.ProviderName())),
	}
}

// See [finance.APIProvider.ProviderName]
func (p *cachedFXProvider) ProviderName() string {
	return p.provider.ProviderName()
}

// See [finance.APIProvider.Shutdown]
func (p *cachedFXProvider) Shutdown(ctx context.Context) error {
	return p.provider.Shutdown(ctx)
}

// See [finance.APIProvider.Close]
func (p *cachedFXProvider) Close() error {
	return p.provider.Close()
}

// See [finance.APIProvider.QueryExchangeRate]
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
	} else {
		p.logger.Info("using cached exchange rate", slog.String("base", string(base)), slog.String("quote", string(quote)), slog.Float64("rate", cachedExchangeRate.Rate))
	}
	return cachedExchangeRate, nil
}

func (p *cachedFXProvider) cacheKey(base, quote finance.Currency) string {
	// Deliberately NOT provider scoped, unlike the listing cache: the FX providers
	// form a fallback chain that is supposed to yield exactly one authoritative
	// rate. With a per-provider key each provider would cache its own rate, so a
	// caller would get the ECB rate on one call and the streaming rate on the next,
	// depending on which provider happened to answer. Provenance stays correct
	// either way — the cached ExchangeRate carries its own Source.
	return fmt.Sprintf("finance:exchangeRate:%s/%s", strings.ToUpper(string(base)), strings.ToUpper(string(quote)))
}

type FallbackFXProvider struct {
	queue *cooldownQueue[finance.FX]
}

func NewFallbackFXProvider() *FallbackFXProvider {
	return &FallbackFXProvider{
		queue: newCooldownQueue[finance.FX](),
	}
}

// Add adds the given [finance.FX] provider to the provider's queue.
func (p *FallbackFXProvider) Add(provider finance.FX, cooldownTTL time.Duration) {
	p.queue.Add(provider, cooldownTTL)
}

// See [finance.APIProvider.ProviderName]
func (p *FallbackFXProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("fallback:")
	initialBufferLen := buffer.Len()
	p.queue.ForEach(func(provider finance.FX, _ error) {
		if buffer.Len() > initialBufferLen {
			buffer.WriteRune('|')
		}
		buffer.WriteString(provider.ProviderName())
	})
	return buffer.String()
}

// See [finance.APIProvider.Shutdown]
func (p *FallbackFXProvider) Shutdown(ctx context.Context) error {
	errs := make([]error, 0)
	p.queue.ForEach(func(provider finance.FX, _ error) {
		err := provider.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}

// See [finance.APIProvider.Close]
func (p *FallbackFXProvider) Close() error {
	errs := make([]error, 0)
	p.queue.ForEach(func(provider finance.FX, _ error) {
		err := provider.Close()
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}

// See [finance.FX.QueryExchangeRate]
func (p *FallbackFXProvider) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	var rate *finance.ExchangeRate
	errs := make([]error, 0)
	p.queue.ForEach(func(provider finance.FX, lastErr error) {
		if rate == nil && lastErr == nil {
			providerRate, err := provider.QueryExchangeRate(ctx, base, quote)
			if err == nil {
				rate = providerRate
			} else {
				errs = append(errs, err)
				if !finance.IsTransientErr(err) {
					p.queue.MarkProviderFailed(provider, err)
				}
			}
		}
	})
	if rate == nil {
		if slices.Contains(errs, finance.ErrRequestPending) {
			return nil, finance.ErrRequestPending
		} else if len(errs) == 0 {
			return nil, finance.ErrExchangeRateNotAvailable
		}
		return nil, fmt.Errorf("%w: %w", finance.ErrExchangeRateNotAvailable, errors.Join(errs...))
	}
	return rate, nil
}
