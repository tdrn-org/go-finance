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
	_ "embed"
	"encoding/json"
	"log/slog"

	"github.com/tdrn-org/go-finance"
)

//go:embed stockExchangeMap.json
var stockExchangeMapData []byte

var stockExchangeMap map[string]string = func() map[string]string {
	var stockExchangeMap map[string]string
	err := json.Unmarshal(stockExchangeMapData, &stockExchangeMap)
	if err != nil {
		panic(err)
	}
	return stockExchangeMap
}()

// See [finance.InstrumentProvider]
func (api *API) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	securityCodes := queryToSecurityCodes(query)
	if len(securityCodes) == 0 {
		return nil, finance.ErrInstrumentSearchRestricted
	}
	instruments := make([]finance.Instrument, 0)
	for _, securityCode := range securityCodes {
		reply, err := api.getSecurityInfo(ctx, session, securityService, securityCode)
		if err != nil {
			return nil, err
		}
		for _, stockExchangeInfo := range reply.StockExchangeInfos {
			mic, ok := stockExchangeMap[stockExchangeInfo.StockExchange.Id]
			if !ok {
				api.logger.Warn("unrecognized stock exchange id", slog.String("id", stockExchangeInfo.StockExchange.Id))
				continue
			}
			if mic == "" {
				continue
			}
			instrument := securityInfoToInstrument(reply, mic)
			err = instrument.Validate()
			if err == nil {
				instruments = append(instruments, *instrument)
			} else {
				api.logger.Warn("ignoring invalid instrument", slog.Any("err", err))
			}
		}
	}
	return instruments, nil
}

func (api *API) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	session, err := api.getSessionLocked(ctx)
	if err != nil {
		return nil, err
	}
	securityService := session.SecurityService()
	resolved := make([]finance.Instrument, len(instruments))
	copy(resolved, instruments)
	for i, instrument := range instruments {
		securityCode := instrumentToSecurityCode(&instrument)
		if securityCode == nil {
			continue
		}
		reply, err := api.getSecurityInfo(ctx, session, securityService, securityCode)
		if err != nil {
			return nil, err
		}
		for _, stockExchangeInfo := range reply.StockExchangeInfos {
			mic, _ := stockExchangeMap[stockExchangeInfo.StockExchange.Id]
			if mic == instrument.MIC {
				instrument.Merge(securityInfoToInstrument(reply, mic))
				resolved[i] = instrument
				break
			}
		}
	}
	return resolved, nil
}
