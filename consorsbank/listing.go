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
	"time"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
)

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	logger := api.logger.With(slog.String("query", query))
	logger.Info("search listings...")
	traceCtx, span := api.tracer.StartConsuming(ctx)
	defer span.End()

	api.mutex.Lock()
	defer api.mutex.Unlock()

	listings, err := api.searchListingsLocked(traceCtx, query)
	if errors.Is(err, finance.ErrListingSearchRestricted) {
		logger.Info("ignoriing unsupported listing search")
	} else if err != nil {
		logger.Warn("search listings failure", slog.Any("err", err))
	} else {
		logger.Info("listings received", slog.Int("len", len(listings)))
	}
	return listings, tracehelper.SetStatus(span, err)
}

func (api *API) searchListingsLocked(ctx context.Context, query string) ([]finance.Listing, error) {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCodes := queryToSecurityCodes(query)
	if len(securityCodes) == 0 {
		return []finance.Listing{}, finance.ErrListingSearchRestricted
	}
	listings := make([]finance.Listing, 0)
	for _, securityCode := range securityCodes {
		reply, err := api.getSecurityInfoLocked(ctx, session, securityService, securityCode)
		if err != nil {
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

func (api *API) getSecurityInfoLocked(ctx context.Context, session *apiSession, securityService proto.SecurityServiceClient, securityCode *proto.SecurityCode) (*proto.SecurityInfoReply, error) {
	reply, err := securityService.GetSecurityInfo(ctx, &proto.SecurityInfoRequest{
		AccessToken:  session.AccessToken,
		SecurityCode: securityCode,
	})
	if err != nil {
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("%s: failed to query security info (cause: %w)", Name, err)
	}
	if tapiError := reply.GetError(); tapiError != nil {
		return nil, fmt.Errorf("%s: query security info error (code: %s, message: %s)",
			Name, tapiError.GetCode(), tapiError.GetMessage())
	}
	return reply, nil
}

func (api *API) includeSecurityStockExchange(stockExchangeInfo *proto.SecurityStockExchangeInfo) bool {
	if len(api.includeExchanges) > 0 && !slices.Contains(api.includeExchanges, stockExchangeInfo.StockExchange.Id) {
		return false
	}
	return true
}

// See [finance.ListingProvider.QueryQuote]
func (api *API) QueryQuote(ctx context.Context, listing finance.Listing) (*finance.Quote, error) {
	err := listing.CheckProvider(Name)
	if err != nil {
		return nil, err
	}

	logger := api.logger.With(slog.String("listing", listing.ID()))
	logger.Info("querying quote...")
	traceCtx, span := api.tracer.StartConsuming(ctx)
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
	isin, err := listing.MustAttribute(finance.ListingAttributeKeyISIN)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCode := &proto.SecurityCode{
		Code:     isin,
		CodeType: proto.SecurityCodeType_ISIN,
	}
	securityInfoReply, err := api.getSecurityInfoLocked(ctx, session, securityService, securityCode)
	if err != nil {
		return nil, err
	}
	listingExchange, err := api.getListingExchange(listing, securityInfoReply.StockExchangeInfos)
	if err != nil {
		return nil, err
	}
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := securityService.StreamMarketData(subscriptionCtx, &proto.SecurityMarketDataRequest{
		AccessToken: session.AccessToken,
		SecurityWithStockexchange: &proto.SecurityWithStockExchange{
			SecurityCode:  securityCode,
			StockExchange: listingExchange.StockExchange,
		},
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

func (api *API) getListingExchange(listing finance.Listing, exchangeInfos []*proto.SecurityStockExchangeInfo) (*proto.SecurityStockExchangeInfo, error) {
	exchange, err := listing.MustAttribute(finance.ListingAttributeKeyExchange)
	if err != nil {
		return nil, err
	}
	for _, exchangeInfo := range exchangeInfos {
		if exchangeInfo.StockExchange.Id == exchange {
			return exchangeInfo, nil
		}
	}
	return nil, fmt.Errorf("%s: unable to determine exchange for listing: %s", Name, listing.String())
}

func (api *API) processSecurityMarketDataReply(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], listing finance.Listing, reply *proto.SecurityMarketDataReply) {
	if reply.Error != nil {
		api.logger.Warn("errornous market data reply; closing stream", slog.Any("error", reply.Error))
		s.Close()
		return
	}
	s.RecordReply(securityMarketDataReplyToQuote(listing, reply))
}

func (api *API) securityMarketdataSubscriptionKey(listing finance.Listing) string {
	return listing.ID()
}

// See [finance.ListingProvider.QueryQuoteHistory]
func (api *API) QueryQuoteHistory(ctx context.Context, listing finance.Listing, from, to time.Time) ([]finance.Quote, error) {
	return nil, nil
}
