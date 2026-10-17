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

	"github.com/tdrn-org/go-finance"
	openfigiapi "github.com/tdrn-org/go-finance/openfigi/api"
)

func (api *API) MapListings(ctx context.Context, listings []finance.Listing) (any, error) {
	listingsLen := len(listings)
	mappings := make([][]openfigiapi.FigiResult, listingsLen)
	requestLimit := 5
	if api.apiKey != "" {
		requestLimit = 100
	}
	jobMap := make(map[int]*openfigiapi.MappingJob, requestLimit)
	for i := range listingsLen {
		listing := listings[i]
		job := listingToMappingJob(listing)
		if job != nil {
			jobMap[i] = job
		}
		jobMapLen := len(jobMap)
		if jobMapLen < requestLimit && i+1 < listingsLen {
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
			return nil, fmt.Errorf("%s: unexpected mapping response length: %d", Name, responseLen)
		}
		for k, result := range *response {
			listingIndex := mappedIndex[k]
			figiResult := mappingJobResultToFigiResult(&result)
			mappings[listingIndex] = figiResult
		}
	}
	return mappings, nil
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
