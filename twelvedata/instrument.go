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

package twelvedata

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/tdrn-org/go-finance"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

// See [finance.InstrumentProvider]
func (api *API) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	instruments := make([]finance.Instrument, 0)
	err := api.executeSearch(ctx, query, func(responseItem *twelvedata.SymbolSearchResponseItem) error {
		if !api.includeSymmbolSearchResponseItem(responseItem) {
			return nil
		}
		instrument := symbolSearchResponseItemToInstrument(responseItem)
		err := instrument.Validate()
		if err == nil {
			instruments = append(instruments, *instrument)
		} else {
			api.logger.Warn("ignoring invalid instrument", slog.Any("err", err))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return instruments, nil
}

func (api *API) includeSymmbolSearchResponseItem(responseItem *twelvedata.SymbolSearchResponseItem) bool {
	if len(api.mics) > 0 && !slices.Contains(api.mics, responseItem.MicCode) {
		return false
	}
	if len(api.includeInstrumentTypes) > 0 && !slices.Contains(api.includeInstrumentTypes, responseItem.InstrumentType) {
		return false
	}
	return true
}

// See [finance.InstrumentProvider]
func (api *API) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	resolved := make([]finance.Instrument, len(instruments))
	copy(resolved, instruments)
	for i, instrument := range instruments {
		if instrument.HasIdentifier(finance.InstrumentIdentifierTicker) {
			continue
		}
		query := instrumentToQuery(&instrument)
		if query == "" {
			continue
		}
		err := api.executeSearch(ctx, query, func(responseItem *twelvedata.SymbolSearchResponseItem) error {
			if responseItem.MicCode != instrument.MIC {
				return nil
			}
			instrument.Merge(symbolSearchResponseItemToInstrument(responseItem))
			resolved[i] = instrument
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

func (api *API) executeSearch(ctx context.Context, query string, collect func(*twelvedata.SymbolSearchResponseItem) error) error {
	response, rsp, err := api.client.ReferenceDataAPI.
		GetSymbolSearch(ctx).
		Symbol(query).
		Execute()
	if err != nil {
		return fmt.Errorf("%s: search symbol failure (cause: %w)", Name, err)
	}
	err = api.checkHttpStatus(rsp)
	if err != nil {
		return err
	}
	for _, responseItem := range response.Data {
		err = collect(&responseItem)
		if err != nil {
			return err
		}
	}
	return nil
}
