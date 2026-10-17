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
	"testing"

	"github.com/tdrn-org/go-finance"
)

type stubListingProvider struct {
	name string
}

func (p *stubListingProvider) ProviderName() string { return p.name }

func (p *stubListingProvider) SearchListings(_ context.Context, _ string) ([]finance.Listing, error) {
	return nil, nil
}

func (p *stubListingProvider) QueryQuote(_ context.Context, _ finance.Listing) (*finance.Quote, error) {
	return nil, nil
}

type stubFXProvider struct {
	name string
}

func (p *stubFXProvider) ProviderName() string { return p.name }

func (p *stubFXProvider) QueryExchangeRate(_ context.Context, _, _ finance.Currency) (*finance.ExchangeRate, error) {
	return nil, nil
}

// TestListingCacheKeyIsProviderScoped guards against two providers sharing one
// cache entry for the same query. All cached providers get the same cache
// instance, so without the provider in the key the second provider overwrites the
// first one's result and both then answer with the same list — which showed up in
// production as every instrument being returned twice.
func TestListingCacheKeyIsProviderScoped(t *testing.T) {
	providerA := &cachedListingProvider{provider: &stubListingProvider{name: "provider-a"}}
	providerB := &cachedListingProvider{provider: &stubListingProvider{name: "provider-b"}}

	keyA := providerA.listingCacheKey("US0378331005")
	keyB := providerB.listingCacheKey("US0378331005")
	if keyA == keyB {
		t.Fatalf("cache key is not provider scoped, both providers use %q", keyA)
	}
	if providerA.listingCacheKey("us0378331005") != keyA {
		t.Errorf("cache key is not case insensitive: %q vs %q", providerA.listingCacheKey("us0378331005"), keyA)
	}
}

// TestFXCacheKeyIsProviderScoped guards the same for exchange rates. An ECB
// reference rate and a streaming live rate must never occupy the same entry: the
// caller would get either value depending on who filled the cache first, while the
// source field of the returned rate would claim the other provider.
func TestFXCacheKeyIsProviderScoped(t *testing.T) {
	providerA := &cachedFXProvider{provider: &stubFXProvider{name: "frankfurter"}}
	providerB := &cachedFXProvider{provider: &stubFXProvider{name: "consorsbank"}}

	keyA := providerA.cacheKey(finance.CurrencyUSD, finance.CurrencyEUR)
	keyB := providerB.cacheKey(finance.CurrencyUSD, finance.CurrencyEUR)
	if keyA == keyB {
		t.Fatalf("cache key is not provider scoped, both providers use %q", keyA)
	}
	if providerA.cacheKey(finance.CurrencyUSD, finance.CurrencyEUR) != keyA {
		t.Errorf("cache key is not deterministic")
	}
	if providerA.cacheKey(finance.CurrencyEUR, finance.CurrencyUSD) == keyA {
		t.Errorf("cache key does not distinguish base from quote")
	}
}
