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
	"fmt"
	"log/slog"
	"slices"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
)

// See [finance.ListingProvider.SameListing]
func (api *API) SameListing(listing1, listing2 finance.ProviderListing) bool {
	return finance.SameListing(listing1, listing2, Name, finance.ListingAttributeKeyISIN, finance.ListingAttributeKeyExchange)
}

// See [finance.ListingProvider.SearchListings]
func (api *API) SearchListings(ctx context.Context, query string) ([]finance.ProviderListing, error) {
	api.logger.Info("search listings...", slog.String("query", query))

	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCodes := queryToSecurityCodes(query)
	if len(securityCodes) == 0 {
		return nil, finance.ErrListingSearchRestricted
	}
	listings := make([]finance.ProviderListing, 0)
	for _, securityCode := range securityCodes {
		reply, err := api.getSecurityInfo(ctx, session, securityService, securityCode)
		if err != nil {
			return nil, err
		}
		for _, stockExchangeInfo := range reply.StockExchangeInfos {
			if !api.includeSecurityStockExchange(stockExchangeInfo) {
				continue
			}
			listing, err := securityInfoToProviderListing(reply, stockExchangeInfo)
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
func (api *API) QueryQuote(ctx context.Context, listing finance.ProviderListing) (*finance.Quote, error) {
	err := listing.CheckProvider(Name)
	if err != nil {
		return nil, err
	}

	api.logger.Info("querying quote...", slog.Any("listing", listing))

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
	return subscription.LastReply()
}

func (api *API) getSecurityInfo(ctx context.Context, session *apiSession, securityService proto.SecurityServiceClient, securityCode *proto.SecurityCode) (*proto.SecurityInfoReply, error) {
	reply, err := securityService.GetSecurityInfo(ctx, &proto.SecurityInfoRequest{
		AccessToken:  session.AccessToken,
		SecurityCode: securityCode,
	})
	if err != nil {
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("%s: failed to query security info from TAPI at %q (cause: %w)", Name, api.address, err)
	}
	if tapiError := reply.GetError(); tapiError != nil {
		return nil, fmt.Errorf("%s: query security info error in call to TAPI at %q (code: %s, message: %s)",
			Name, api.address, tapiError.GetCode(), tapiError.GetMessage())
	}
	return reply, nil
}

func (api *API) getSecurityMarketDataSubscriptionLocked(listing finance.ProviderListing) *streamSubscription[proto.SecurityMarketDataReply, finance.Quote] {
	subscriptionKey := listing.String()
	subscription := api.quoteSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startSecurityMarketDataSubscriptionLocked(ctx context.Context, listing finance.ProviderListing) (*streamSubscription[proto.SecurityMarketDataReply, finance.Quote], error) {
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
	securityInfoReply, err := api.getSecurityInfo(ctx, session, securityService, securityCode)
	if err != nil {
		return nil, err
	}
	if len(securityInfoReply.StockExchangeInfos) == 0 {
		return nil, fmt.Errorf("%s: unable to determine stock exchange for quote query (code: %s)", Name, securityCode.Code)
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
		//		Currency: string(api.preferredCurrency),
		Currency: "USD",
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("%s: failed to create security market data subscription (cause: %w)", Name, err)
	}
	subscriptionKey := listing.String()
	subscription := &streamSubscription[proto.SecurityMarketDataReply, finance.Quote]{
		client:              client,
		subscriptionTimeout: api.subscriptionTimeout,
		recordReply: func(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], reply *proto.SecurityMarketDataReply) {
			recordSecurityMarketDataReply(s, listing, reply)
		},
		ctx:    subscriptionCtx,
		cancel: subscriptionCancel,
		logger: api.logger.With(slog.String("securityMarketDataSubscription", subscriptionKey)),
	}
	api.quoteSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
}

func (api *API) getListingExchange(listing finance.ProviderListing, exchangeInfos []*proto.SecurityStockExchangeInfo) (*proto.SecurityStockExchangeInfo, error) {
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

func recordSecurityMarketDataReply(s *streamSubscription[proto.SecurityMarketDataReply, finance.Quote], listing finance.ProviderListing, reply *proto.SecurityMarketDataReply) {
	if reply.Error != nil {
		s.logger.Warn("ignoring errornous market data reply", slog.Any("err", reply.Error))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("recording security market data reply")
	s.lastReply = securityMarketDataReplyToQuote(listing, reply)
}
