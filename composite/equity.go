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
	"strings"

	"github.com/tdrn-org/go-cache"
	"github.com/tdrn-org/go-finance"
)

type QuoteCache cache.KeyValue[string, *finance.Quote]

type equityCache struct {
	provider finance.Equity
	cache    QuoteCache
}

func NewCachedEquityProvider(provider finance.Equity, cache QuoteCache) finance.Equity {
	return &equityCache{
		provider: provider,
		cache:    cache,
	}
}

func (p *equityCache) ProviderName() string {
	buffer := &strings.Builder{}
	buffer.WriteString("cached:")
	buffer.WriteString(p.provider.ProviderName())
	return buffer.String()
}

func (p *equityCache) QueryQuote(ctx context.Context, symbol finance.Symbol) (*finance.Quote, error) {
	key := fmt.Sprintf("finance:quote:%s/%s/%s/%s/%s", symbol.Exchange, symbol.Ticker, symbol.ISIN, symbol.WKN, symbol.FIGI)
	cachedQuote, err := p.cache.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		cachedQuote, err = p.provider.QueryQuote(ctx, symbol)
		if err != nil {
			return nil, err
		}
		p.cache.Put(ctx, key, cachedQuote)
	} else if err != nil {
		return nil, err
	}
	return cachedQuote, nil
}
