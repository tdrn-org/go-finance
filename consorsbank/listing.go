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

package consorsbank

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
	"go.opentelemetry.io/otel/attribute"
)

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	logger := api.logger.With(slog.String("query", query))
	logger.Info("search listings...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.String("query", query))
	defer span.End()

	listings, err := api.searchListings(traceCtx, query)
	if errors.Is(err, finance.ErrListingSearchRestricted) {
		logger.Info("ignoring unsupported listing search")
	} else if err != nil {
		logger.Warn("search listings failure", slog.Any("err", err))
	} else {
		logger.Info("listings received", slog.Int("len", len(listings)))
	}
	return listings, tracehelper.SetStatus(span, err)
}

func (api *API) searchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityCodes := queryToSecurityCodes(query)
	if len(securityCodes) == 0 {
		return []finance.Listing{}, finance.ErrListingSearchRestricted
	}
	listings := make([]finance.Listing, 0)
	for _, securityCode := range securityCodes {
		reply, err := api.getSecurityInfoLocked(ctx, session, securityCode)
		if err != nil {
			api.invalidateSessionLocked()
			return nil, err
		}
		for _, stockExchangeInfo := range reply.StockExchangeInfos {
			if !api.includeSecurityStockExchange(stockExchangeInfo) {
				continue
			}
			listing, err := securityInfoToListing(reply, stockExchangeInfo)
			if err == nil {
				listings = append(listings, *listing)
			} else {
				api.logger.Warn("ignoring invalid listing", slog.Any("err", err))
			}
		}
	}
	return listings, nil
}

func (api *API) includeSecurityStockExchange(stockExchangeInfo *proto.SecurityStockExchangeInfo) bool {
	if len(api.includeExchanges) > 0 && !slices.Contains(api.includeExchanges, stockExchangeInfo.StockExchange.Id) {
		return false
	}
	return true
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()))
	logger.Info("querying quote...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.String("listing", listing.ID()))
	defer span.End()

	quote, err := api.queryQuote(traceCtx, listing)
	if errors.Is(err, finance.ErrRequestPending) {
		logger.Info("quote request pending")
	} else if err != nil {
		logger.Warn("query quote failure", slog.Any("err", err))
	} else {
		logger.Info("quote received", slog.Float64("price", quote.Price), slog.String("currency", string(quote.Currency)))
	}
	return quote, tracehelper.SetStatus(span, err)
}

func (api *API) queryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	subscription, err := api.getOrStartSecurityMarketDataSubscription(ctx, listing)
	if err != nil {
		return nil, err
	}
	return subscription.waitInitialReply(ctx, initialReplyTimeout)
}

func (api *API) getOrStartSecurityMarketDataSubscription(ctx context.Context, listing finance.Listing) (*streamSubscription[proto.SecurityMarketDataReply, finance.Quote], error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	subscription := api.getSecurityMarketDataSubscriptionLocked(listing)
	if subscription == nil {
		var err error
		subscription, err = api.startSecurityMarketDataSubscriptionLocked(ctx, listing)
		if err != nil {
			return nil, err
		}
	}
	return subscription, nil
}

func (api *API) getSecurityMarketDataSubscriptionLocked(listing finance.Listing) *streamSubscription[proto.SecurityMarketDataReply, finance.Quote] {
	subscriptionKey := api.securityMarketdataSubscriptionKey(listing)
	subscription := api.quoteSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startSecurityMarketDataSubscriptionLocked(ctx context.Context, listing finance.Listing) (*streamSubscription[proto.SecurityMarketDataReply, finance.Quote], error) {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityCode, exchange, err := listingToSecurityCodeAndExchange(listing)
	if err != nil {
		return nil, err
	}
	securityWithStockExchange, err := api.getSecurityWithStockExchangeLocked(ctx, session, securityCode, exchange)
	if err != nil {
		api.invalidateSessionLocked()
		return nil, err
	}
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := session.SecurityService.StreamMarketData(subscriptionCtx, &proto.SecurityMarketDataRequest{
		AccessToken:               session.AccessToken,
		SecurityWithStockexchange: securityWithStockExchange,
		// The TAPI converts into the requested currency, so a US instrument can be
		// quoted in EUR directly — no exchange rate conversion needed downstream. An
		// empty value leaves the field unset and the TAPI answers in the instrument's
		// default currency.
		Currency: string(api.preferredCurrency),
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("%s: failed to create security market data subscription (cause: %w)", Name, err)
	}
	subscriptionKey := api.securityMarketdataSubscriptionKey(listing)
	processReply := func(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], reply *proto.SecurityMarketDataReply) {
		api.processSecurityMarketDataReply(s, listing, reply)
	}
	subscription := newSubscription(client, subscriptionCancel, processReply, api.logger.With(slog.String("securityMarketDataSubscription", subscriptionKey)))
	subscription.Timeout = api.subscriptionTimeout
	api.quoteSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
}

func (api *API) processSecurityMarketDataReply(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], listing finance.Listing, reply *proto.SecurityMarketDataReply) {
	if reply.Error != nil {
		api.logger.Warn("errornous market data reply; closing stream", slog.Any("error", reply.Error))
		s.Close()
		return
	}
	s.MergeReply(reply, func(lastReply *finance.Quote, rawReply *proto.SecurityMarketDataReply) *finance.Quote {
		return securityMarketDataReplyToQuote(listing, lastReply, reply)
	})
}

func (api *API) securityMarketdataSubscriptionKey(listing finance.Listing) string {
	return listing.ID()
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (api *API) QueryQuoteHistory(ctx context.Context, listing finance.Listing, history finance.History) (finance.Bars, error) {
	err := listing.CheckProviderName(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()), slog.String("history", string(history)))
	logger.Info("querying quote history...")
	traceCtx, span := api.tracer.StartConsuming(ctx, attribute.String("listing", listing.ID()), attribute.String("history", string(history)))
	defer span.End()

	bars, err := api.queryQuoteHistory(traceCtx, listing, history)
	if errors.Is(err, finance.ErrRequestPending) {
		logger.Info("quote history request pending")
	} else if err != nil {
		logger.Warn("query quote history failure", slog.Any("err", err))
	} else {
		logger.Info("quote history received", slog.Int("len", len(bars)))
	}
	return bars, tracehelper.SetStatus(span, err)
}

func (api *API) queryQuoteHistory(ctx context.Context, listing finance.Listing, history finance.History) (finance.Bars, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityCode, exchange, err := listingToSecurityCodeAndExchange(listing)
	if err != nil {
		return nil, err
	}
	securityWithStockExchange, err := api.getSecurityWithStockExchangeLocked(ctx, session, securityCode, exchange)
	if err != nil {
		api.invalidateSessionLocked()
		return nil, err
	}
	reply, err := api.getSecurityPriceHistory(ctx, session, securityWithStockExchange, history)
	if err != nil {
		api.invalidateSessionLocked()
		return nil, err
	}
	return securityPriceHistoryReplyToBars(listing, reply), nil
}
