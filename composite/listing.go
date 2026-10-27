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

type ListingCache cache.KeyValue[string, []finance.Listing]
type QuoteCache cache.KeyValue[string, *finance.Quote]

type cachedListingProvider struct {
	provider     finance.ListingProvider
	listingCache ListingCache
	quoteCache   QuoteCache
	logger       *slog.Logger
}

func NewCachedListingProvider(provider finance.ListingProvider, listingCache ListingCache, quoteCache QuoteCache) finance.ListingProvider {
	return &cachedListingProvider{
		provider:     provider,
		listingCache: listingCache,
		quoteCache:   quoteCache,
		logger:       slog.With(slog.String("provider", provider.ProviderName())),
	}
}

// See [finance.APIProvider.ProviderName]
func (p *cachedListingProvider) ProviderName() string {
	return p.provider.ProviderName()
}

// See [finance.APIProvider.Shutdown]
func (p *cachedListingProvider) Shutdown(ctx context.Context) error {
	return p.provider.Shutdown(ctx)
}

// See [finance.APIProvider.Close]
func (p *cachedListingProvider) Close() error {
	return p.provider.Close()
}

// See [finance.ListingProvider.SearchListings]
func (p *cachedListingProvider) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	key := p.listingCacheKey(query)
	cachedListings, err := p.listingCache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedListings, err = p.provider.SearchListings(ctx, query)
		if err != nil {
			return nil, err
		}
		p.listingCache.Put(ctx, key, cachedListings)
	} else if err != nil {
		return nil, err
	} else {
		p.logger.Info("using cached listings", slog.Int("len", len(cachedListings)))
	}
	return cachedListings, nil
}

func (p *cachedListingProvider) listingCacheKey(query string) string {
	// The provider name is part of the key because all cached providers share one
	// cache instance. Without it two providers would return each other's results:
	// the second provider overwrites the entry, and from then on both answer with
	// the same list (visible as every instrument appearing twice).
	return fmt.Sprintf("finance:listings:%s:%s", p.provider.ProviderName(), strings.ToUpper(query))
}

// See [finance.ListingProvider.QueryQuote]
func (p *cachedListingProvider) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	key := p.quoteCacheKey(listing)
	cachedQuote, err := p.quoteCache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedQuote, err = p.provider.QueryQuote(ctx, listing)
		if err != nil {
			return nil, err
		}
		p.quoteCache.Put(ctx, key, cachedQuote)
	} else if err != nil {
		return nil, err
	} else {
		p.logger.Info("using cached quote", slog.String("listing", cachedQuote.Listing.ID()), slog.Float64("price", cachedQuote.Price))
	}
	return cachedQuote, nil
}

func (p *cachedListingProvider) quoteCacheKey(listing finance.Listing) string {
	return fmt.Sprintf("finance:quote:%s:%s", p.provider.ProviderName(), listing.ID())
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (p *cachedListingProvider) QueryQuoteHistory(ctx context.Context, listing finance.Listing, from, to time.Time) ([]finance.Quote, error) {
	return nil, nil
}

type CombinedListingProvider struct {
	queue *cooldownQueue[finance.ListingProvider]
}

func NewCombinedListingProvider() *CombinedListingProvider {
	return &CombinedListingProvider{
		queue: newCooldownQueue[finance.ListingProvider](),
	}
}

// Add adds the given [finance.ListingProvider] provider to the provider's queue.
func (p *CombinedListingProvider) Add(provider finance.ListingProvider, cooldownTTL time.Duration) {
	p.queue.Add(provider, cooldownTTL)
}

// See [finance.APIProvider.ProviderName]
func (p *CombinedListingProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("combined:")
	initialBufferLen := buffer.Len()
	p.queue.ForEach(func(provider finance.ListingProvider, _ error) {
		if buffer.Len() > initialBufferLen {
			buffer.WriteRune('|')
		}
		buffer.WriteString(provider.ProviderName())
	})
	return buffer.String()
}

// See [finance.APIProvider.Shutdown]
func (p *CombinedListingProvider) Shutdown(ctx context.Context) error {
	errs := make([]error, 0)
	p.queue.ForEach(func(provider finance.ListingProvider, _ error) {
		err := provider.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}

// See [finance.APIProvider.Close]
func (p *CombinedListingProvider) Close() error {
	errs := make([]error, 0)
	p.queue.ForEach(func(provider finance.ListingProvider, _ error) {
		err := provider.Close()
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}

// See [finance.ListingProvider.SearchListings]
func (p *CombinedListingProvider) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	listings := make([]finance.Listing, 0)
	p.queue.ForEach(func(provider finance.ListingProvider, lastErr error) {
		if lastErr == nil {
			providerListings, err := provider.SearchListings(ctx, query)
			if err == nil {
				listings = append(listings, providerListings...)
			} else if !errors.Is(err, finance.ErrListingSearchRestricted) {
				p.queue.MarkProviderFailed(provider, err)
			}
		}
	})
	return listings, nil
}

// See [finance.ListingProvider.QueryQuote]
func (p *CombinedListingProvider) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	var quote *finance.Quote
	var err error
	p.queue.ForEach(func(provider finance.ListingProvider, lastErr error) {
		if quote == nil && err == nil {
			err := listing.CheckProvider(provider)
			if err == nil {
				if lastErr != nil {
					err = lastErr
				} else {
					quote, err = provider.QueryQuote(ctx, listing)
					if err != nil {
						p.queue.MarkProviderFailed(provider, err)
					}
				}
			}
		}
	})
	return quote, err
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (p *CombinedListingProvider) QueryQuoteHistory(ctx context.Context, listing finance.Listing, from, to time.Time) ([]finance.Quote, error) {
	return nil, nil
}
