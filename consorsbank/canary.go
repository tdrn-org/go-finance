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
)

const canaryISIN string = "DE0007164600"
const canaryExchange string = "TRG"
const canaryTimeout time.Duration = 10 * time.Minute

func (api *API) newCanarySubscriptionLocked(ctx context.Context, session *apiSession) (*streamSubscription[proto.SecurityMarketDataReply, time.Time], error) {
	securityCode := &proto.SecurityCode{
		Code:     canaryISIN,
		CodeType: proto.SecurityCodeType_ISIN,
	}
	securityWithStockExchange, err := api.getSecurityWithStockExchangeLocked(ctx, session, securityCode, canaryExchange)
	if err != nil {
		return nil, err
	}
	subscriptionCtx, subscriptionCancel := context.WithCancel(context.Background())
	client, err := session.SecurityService.StreamMarketData(subscriptionCtx, &proto.SecurityMarketDataRequest{
		AccessToken:               session.AccessToken,
		SecurityWithStockexchange: securityWithStockExchange,
		Currency:                  string(api.preferredCurrency),
	})
	if err != nil {
		subscriptionCancel()
		return nil, fmt.Errorf("%s: failed to create canary subscription (cause: %w)", Name, err)
	}
	canaryLogger := api.logger.With(slog.String("canarySubscription", canaryISIN+"/"+canaryExchange))
	subscription := newSubscription(client, subscriptionCancel, api.processCanarySecurityMarketDataReply, canaryLogger)
	api.stoppedWG.Go(subscription.Start)
	canaryLogger.Info("canary subscription initiated...")
	return subscription, nil
}

func (api *API) processCanarySecurityMarketDataReply(s *streamSubscription[proto.SecurityMarketDataReply, time.Time], reply *proto.SecurityMarketDataReply) {
	s.MergeReply(reply, func(lastReply *time.Time, rawReply *proto.SecurityMarketDataReply) (*time.Time, error) {
		merged := lastReply
		if slices.Contains(rawReply.ChangedFields, proto.SecurityMarketDataReply_LAST_DATE_TIME) {
			lastDateTime := time.Unix(rawReply.LastDateTime.Seconds, int64(rawReply.LastDateTime.Nanos)).UTC()
			merged = &lastDateTime
		}
		return merged, nil
	})
}

func (api *API) checkCanarySubscription() error {
	session := api.getSession()
	if session == nil {
		return ErrSessionClosed
	}
	lastReply, lastReplyErr := session.canarySubscription.LastReply()
	if errors.Is(lastReplyErr, finance.ErrRequestPending) {
		lastReply = &session.CreatedAt
	} else if lastReplyErr != nil {
		return lastReplyErr
	}
	elapsed := TradegateTradingDurationSince(*lastReply)
	if elapsed > canaryTimeout {
		return finance.ErrSubscriptionStalled
	}
	return nil
}
