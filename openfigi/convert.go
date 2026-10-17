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
	_ "embed"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/openfigi/api"
)

func listingToMappingJob(listing finance.Listing) *api.MappingJob {
	var idType api.MappingJobIdType
	idValue := api.MappingJob_IdValue{}
	var micCode *string

	isin, hasISIN := listing.Attribute(finance.ListingAttributeKeyISIN)
	wkn, hasWKN := listing.Attribute(finance.ListingAttributeKeyWKN)
	ticker, hasTicker := listing.Attribute(finance.ListingAttributeKeyTicker)
	if hasISIN {
		idType = api.IDISIN
		idValue.FromMappingJobIdValue0(isin)
	} else if hasWKN {
		idType = api.IDWERTPAPIER
		idValue.FromMappingJobIdValue0(wkn)
	} else if hasTicker {
		idType = api.TICKER
		idValue.FromMappingJobIdValue0(ticker)
	} else {
		return nil
	}
	mic, hasMIC := listing.Attribute(finance.ListingAttributeKeyMIC)
	if hasMIC {
		micCode = &mic
	}
	mappingJob := &api.MappingJob{
		IdType:  idType,
		IdValue: idValue,
		MicCode: micCode,
	}
	return mappingJob
}

func mappingJobResultToFigiResult(result *api.MappingJobResult) []api.FigiResult {
	figiList, err := result.AsMappingJobResultFigiList()
	if err != nil || figiList.Data == nil {
		return nil
	}
	return *figiList.Data
}
