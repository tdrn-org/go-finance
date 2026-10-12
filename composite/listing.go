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

type ListingCache cache.KeyValue[string, []finance.ProviderListing]
type QuoteCache cache.KeyValue[string, *finance.Quote]

type cachedListingProvider struct {
	provider     finance.ListingProvider
	listingCache ListingCache
	quoteCache   QuoteCache
}

func NewCachedListingProvider(provider finance.ListingProvider, listingCache ListingCache, quoteCache QuoteCache) finance.ListingProvider {
	return &cachedListingProvider{
		provider:     provider,
		listingCache: listingCache,
		quoteCache:   quoteCache,
	}
}

func NewCachedListingProviders(providers []finance.ListingProvider, listingCache ListingCache, quoteCache QuoteCache) []finance.ListingProvider {
	cachedProviders := make([]finance.ListingProvider, 0, len(providers))
	for _, provider := range providers {
		cachedProviders = append(cachedProviders, NewCachedListingProvider(provider, listingCache, quoteCache))
	}
	return cachedProviders
}

// See [finance.APIProvider.ProviderName]
func (p *cachedListingProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("cached:")
	buffer.WriteString(p.provider.ProviderName())
	return buffer.String()
}

// See [finance.ListingProvider.SameListing]
func (p *cachedListingProvider) SameListing(listing1, listing2 finance.ProviderListing) bool {
	return p.provider.SameListing(listing1, listing2)
}

// See [finance.ListingProvider.SearchListings]
func (p *cachedListingProvider) SearchListings(ctx context.Context, query string) ([]finance.ProviderListing, error) {
	key := p.listingCacheKey(query)
	cachedInstruments, err := p.listingCache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedInstruments, err = p.provider.SearchListings(ctx, query)
		if err != nil {
			return nil, err
		}
		p.listingCache.Put(ctx, key, cachedInstruments)
	} else if err != nil {
		return nil, err
	}
	return cachedInstruments, nil
}

func (p *cachedListingProvider) listingCacheKey(query string) string {
	return fmt.Sprintf("finance:listings:%s", strings.ToUpper(query))
}

// See [finance.ListingProvider.QueryQuote]
func (p *cachedListingProvider) QueryQuote(ctx context.Context, listing finance.ProviderListing) (*finance.Quote, error) {
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
	}
	return cachedQuote, nil
}

func (p *cachedListingProvider) quoteCacheKey(listing finance.ProviderListing) string {
	buffer := &strings.Builder{}
	buffer.WriteString("finance:quote:")
	buffer.WriteString(listing.String())
	return buffer.String()
}

type combinedListingProvider struct {
	queue *cooldownQueue[finance.ListingProvider]
}

func NewCombinedListingProvider(cooldown time.Duration, providers ...finance.ListingProvider) finance.ListingProvider {
	return &combinedListingProvider{
		queue: newCooldownQueue(cooldown, providers...),
	}
}

func (p *combinedListingProvider) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("combined:")
	initialBufferLen := buffer.Len()
	p.queue.ForEach(func(provider finance.ListingProvider) {
		if buffer.Len() > initialBufferLen {
			buffer.WriteRune('|')
		}
		buffer.WriteString(provider.ProviderName())
	})
	return buffer.String()
}

// See [finance.ListingProvider.SameListing]
func (p *combinedListingProvider) SameListing(listing1, listing2 finance.ProviderListing) bool {
	// fail fast if provider do not match
	if listing1.Provider() != listing2.Provider() {
		return false
	}
	same := false
	p.queue.ForEach(func(provider finance.ListingProvider) {
		same = same || provider.SameListing(listing1, listing2)
	})
	return same
}

// See [finance.ListingProvider.SearchListings]
func (p *combinedListingProvider) SearchListings(ctx context.Context, query string) ([]finance.ProviderListing, error) {
	availableProviders := p.queue.GetAvailableProviders()
	listings := make([]finance.ProviderListing, 0)
	for _, availableProvider := range availableProviders {
		foundListings, err := availableProvider.SearchListings(ctx, query)
		if errors.Is(err, finance.ErrListingSearchRestricted) {
			continue
		} else if err != nil {
			slog.Warn("marking listing provider as failed", slog.String("provider", availableProvider.ProviderName()), slog.Any("err", err))
			p.queue.MarkProviderFailed(availableProvider)
			continue
		}
		listings = append(listings, foundListings...)
	}
	return listings, nil
}

// See [finance.ListingProvider.QueryQuote]
func (p *combinedListingProvider) QueryQuote(ctx context.Context, listing finance.ProviderListing) (*finance.Quote, error) {
	for _, availableProvider := range p.queue.GetAvailableProviders() {
		err := listing.CheckProvider(availableProvider.ProviderName())
		if err != nil {
			continue
		}
		quote, err := availableProvider.QueryQuote(ctx, listing)
		if err != nil {
			slog.Warn("marking listing provider as failed", slog.String("provider", availableProvider.ProviderName()), slog.Any("err", err))
			p.queue.MarkProviderFailed(availableProvider)
		}
		return quote, err
	}
	return nil, finance.ErrListingProviderMismatch
}
