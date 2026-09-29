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

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
)

// See [finance.FX]
func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	subscription := api.getExchangeRateSubscriptionLocked(base, quote)
	if subscription == nil {
		var err error
		subscription, err = api.startExchangeRateSubscriptionLocked(ctx, base, quote)
		if err != nil {
			return nil, err
		}
	}
	return subscription.LastReply()
}

func (api *API) getExchangeRateSubscriptionLocked(base, quote finance.Currency) *streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate] {
	subscriptionKey := fmt.Sprintf("%s/%s", base, quote)
	subscription := api.exchangeRateSubscriptions[subscriptionKey]
	if subscription.IsClosed() {
		return nil
	}
	return subscription
}

func (api *API) startExchangeRateSubscriptionLocked(ctx context.Context, base, quote finance.Currency) (*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate], error) {
	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	subscriptionKey := fmt.Sprintf("%s/%s", base, quote)
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := securityService.StreamCurrencyRate(subscriptionCtx, &proto.CurrencyRateRequest{
		AccessToken:  session.AccessToken,
		CurrencyFrom: string(base),
		CurrencyTo:   string(quote),
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("failed to create exchange rate subscription (cause: %w)", err)
	}
	subscription := &streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate]{
		client:              client,
		subscriptionTimeout: api.subscriptionTimeout,
		recordReply:         recordCurrencyRateReply,
		ctx:                 subscriptionCtx,
		cancel:              subscriptionCancel,
		logger:              api.logger.With(slog.String("currencyRateSubscription", subscriptionKey)),
	}
	api.exchangeRateSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
}
