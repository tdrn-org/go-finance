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
	"math"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/consorsbank/proto"
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
)

// See [finance.FX.QueryExchangeRate]
func (api *API) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	logger := api.logger.With(slog.String("base", string(base)), slog.String("quote", string(quote)))
	logger.Info("querying exchange rate...")
	traceCtx, span := api.tracer.StartConsuming(ctx)
	defer span.End()

	rate, err := api.queryExchangeRate(traceCtx, base, quote)
	if errors.Is(err, finance.ErrRequestPending) {
		logger.Info("exchange rate request pending")
	} else if err != nil {
		logger.Info("query exchange rate failure", slog.Any("err", err))
	} else {
		logger.Info("exchange rate received", slog.Float64("rate", rate.Rate))
	}
	return rate, tracehelper.SetStatus(span, err)
}

func (api *API) queryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	subscription, err := api.getOrStartExchangeRateSubscription(ctx, base, quote)
	if err != nil {
		return nil, err
	}
	return subscription.waitInitialReply(ctx, initialReplyTimeout)
}

func (api *API) getOrStartExchangeRateSubscription(ctx context.Context, base, quote finance.Currency) (*streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate], error) {
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
	return subscription, nil
}

func (api *API) getExchangeRateSubscriptionLocked(base, quote finance.Currency) *streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate] {
	subscriptionKey := api.exchangeRateSubscriptionKey(base, quote)
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
	subscriptionKey := api.exchangeRateSubscriptionKey(base, quote)
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := securityService.StreamCurrencyRate(subscriptionCtx, &proto.CurrencyRateRequest{
		AccessToken:  session.AccessToken,
		CurrencyFrom: string(base),
		CurrencyTo:   string(quote),
	})
	if err != nil {
		subscriptionCancel()
		api.invalidateSessionLocked()
		return nil, fmt.Errorf("%s: failed to create exchange rate subscription (cause: %w)", Name, err)
	}
	subscription := newSubscription(client, subscriptionCancel, api.processCurrencyRateReply, api.logger.With(slog.String("currencyRateSubscription", subscriptionKey)))
	subscription.Timeout = api.subscriptionTimeout
	api.exchangeRateSubscriptions[subscriptionKey] = subscription
	api.stoppedWG.Go(subscription.Run)
	return subscription, nil
}

func (api *API) processCurrencyRateReply(s *streamSubscription[proto.CurrencyRateReply, finance.ExchangeRate], reply *proto.CurrencyRateReply) {
	if reply.Error != nil {
		api.logger.Warn("ignoring errornous currency rate reply; closing stream", slog.Any("err", reply.Error))
		s.Close()
		return
	}
	if reply.CurrencyFrom == "" || reply.CurrencyTo == "" || math.IsNaN(reply.CurrencyRate) {
		api.logger.Warn("ignoring empty currency rate reply; closing stream")
		s.Close()
		return
	}
	s.MergeReply(reply, currencyRateReplyToExchangeRate)
}

func (api *API) exchangeRateSubscriptionKey(base, quote finance.Currency) string {
	return fmt.Sprintf("%s/%s", base, quote)
}
