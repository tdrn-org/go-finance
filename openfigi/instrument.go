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

package openfigi

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/tdrn-org/go-finance"
	openfigiapi "github.com/tdrn-org/go-finance/openfigi/api"
)

// See [finance.InstrumentProvider]
func (api *API) SearchInstruments(ctx context.Context, query string) ([]finance.Instrument, error) {
	if len(api.mics) == 0 {
		api.logger.Warn("no MICs configured; no search results")
	}
	instruments := make([]finance.Instrument, 0)
	for _, mic := range api.mics {
		request := &openfigiapi.PostSearchJSONRequestBody{
			Query:   &query,
			MicCode: &mic,
		}
		err := api.runPostSearchWithResponse(ctx, request, func(figiResult *openfigiapi.FigiResult) error {
			if !api.includeFigiResult(figiResult) {
				return nil
			}
			instrument := figiResultToInstrument(figiResult, mic)
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
	}
	return instruments, nil
}

func (api *API) runPostSearchWithResponse(ctx context.Context, request *openfigiapi.PostSearchJSONRequestBody, collect func(*openfigiapi.FigiResult) error) error {
	for {
		response, err := api.postSearchWithResponse(ctx, request)
		if err != nil {
			return err
		}
		for _, figiResult := range *response.Data {
			err = collect(&figiResult)
			if err != nil {
				return err
			}
		}
		next := ""
		if response.Next != nil {
			next = *response.Next
		}
		if next == "" {
			break
		}
		request.Start = &next
	}
	return nil
}

func (api *API) includeFigiResult(figiResult *openfigiapi.FigiResult) bool {
	if len(api.includeSecurityTypes) > 0 && figiResult.SecurityType != nil && slices.Contains(api.includeSecurityTypes, *figiResult.SecurityType) {
		return false
	}
	if len(api.includeSecurityTypes2) > 0 && figiResult.SecurityType2 != nil && slices.Contains(api.includeSecurityTypes2, *figiResult.SecurityType2) {
		return false
	}
	return true
}

func (api *API) postSearchWithResponse(ctx context.Context, request *openfigiapi.PostSearchJSONRequestBody) (*openfigiapi.SearchResponseElement, error) {
	const operation string = "PostSearchWithResponse"
	response, err := api.apiClient.PostSearchWithResponse(ctx, *request, api.authenticateRequest)
	if err != nil {
		return nil, api.wrapSystemError(operation, err)
	}
	err = api.checkAPIResponseWithBody(operation, response.HTTPResponse, response.JSON200)
	if err != nil {
		return nil, err
	}
	if response.JSON200.Error != nil {
		errorMessage := *response.JSON200.Error
		if errorMessage != "" {
			return nil, fmt.Errorf("%s: %s API failure (error: %s)", Name, operation, errorMessage)
		}
	}
	return response.JSON200, nil
}

// See [finance.InstrumentProvider]
func (api *API) ResolveInstruments(ctx context.Context, instruments []finance.Instrument) ([]finance.Instrument, error) {
	instrumentsLen := len(instruments)
	resolved := make([]finance.Instrument, instrumentsLen)
	copy(resolved, instruments)
	requestLimit := 5
	if api.apiKey != "" {
		requestLimit = 100
	}
	jobMap := make(map[int]*openfigiapi.MappingJob, requestLimit)
	for i := range instrumentsLen {
		instrument := instruments[i]
		job := instrumentToMappingJob(&instrument)
		if job != nil {
			jobMap[i] = job
		}
		jobMapLen := len(jobMap)
		if jobMapLen < requestLimit && i+1 < instrumentsLen {
			continue
		}
		request := make(openfigiapi.PostMappingJSONRequestBody, 0, jobMapLen)
		mappedIndex := make([]int, 0, jobMapLen)
		for j, job := range jobMap {
			request = append(request, *job)
			mappedIndex = append(mappedIndex, j)
		}
		response, err := api.postMappingWithResponse(ctx, request)
		if err != nil {
			return nil, err
		}
		responseLen := len(*response)
		if responseLen != jobMapLen {
			return nil, fmt.Errorf("unexpected mapping response length: %d", responseLen)
		}
		for k, result := range *response {
			instrumentIndex := mappedIndex[k]
			instrument := instruments[instrumentIndex]
			figiResult, _ := mappingJobResultToFigiResult(&result)
			resolved[instrumentIndex] = *api.mergeFigiResult(&instrument, figiResult)
		}
	}
	return resolved, nil
}

func (api *API) postMappingWithResponse(ctx context.Context, request openfigiapi.PostMappingJSONRequestBody) (*openfigiapi.BulkMappingJobResult, error) {
	const operation string = "PostMappingWithResponse"
	response, err := api.apiClient.PostMappingWithResponse(ctx, request, api.authenticateRequest)
	if err != nil {
		return nil, api.wrapSystemError(operation, err)
	}
	err = api.checkAPIResponseWithBody(operation, response.HTTPResponse, response.JSON200)
	if err != nil {
		return nil, err
	}
	return response.JSON200, nil
}

func (api *API) mergeFigiResult(instrument *finance.Instrument, figiResult *openfigiapi.FigiResult) *finance.Instrument {
	if figiResult == nil {
		return instrument
	}
	merged := instrument.Clone()
	if figiResult.Ticker != nil && !merged.HasIdentifier(finance.InstrumentIdentifierTicker) {
		merged.Identifiers[finance.InstrumentIdentifierTicker] = *figiResult.Ticker
	}
	if figiResult.Figi != nil && !merged.HasIdentifier(finance.InstrumentIdentifierFIGI) {
		merged.Identifiers[finance.InstrumentIdentifierFIGI] = *figiResult.Figi
	}
	return &merged
}
