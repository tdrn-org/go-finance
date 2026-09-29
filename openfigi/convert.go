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
	"encoding/json"

	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/openfigi/api"
)

//go:embed marketSecDesMap.json
var marketSecDesMapData []byte

var marketSecDesMap map[string]finance.InstrumentType = func() map[string]finance.InstrumentType {
	var marketSecDesMap map[string]finance.InstrumentType
	err := json.Unmarshal(marketSecDesMapData, &marketSecDesMap)
	if err != nil {
		panic(err)
	}
	return marketSecDesMap
}()

//go:embed securityType2Map.json
var securityType2MapData []byte

var securityType2Map map[string]finance.InstrumentType = func() map[string]finance.InstrumentType {
	var securityType2Map map[string]finance.InstrumentType
	err := json.Unmarshal(securityType2MapData, &securityType2Map)
	if err != nil {
		panic(err)
	}
	return securityType2Map
}()

func figiResultToInstrument(figiResult *api.FigiResult, mic string) *finance.Instrument {
	if figiResult == nil {
		return nil
	}
	instrument := finance.NewInstrument()
	instrument.Identifiers[finance.InstrumentIdentifierFIGI] = ptrString(figiResult.Figi)
	instrument.Identifiers[finance.InstrumentIdentifierTicker] = ptrString(figiResult.Ticker)
	instrument.Name = ptrString(figiResult.Name)
	instrument.MIC = mic
	instrumentType, ok := securityType2Map[ptrString(figiResult.SecurityType2)]
	if !ok {
		instrumentType, ok = marketSecDesMap[ptrString(figiResult.MarketSector)]
		if !ok {
			instrumentType = finance.InstrumentTypeUnknown
		}
	}
	instrument.Type = instrumentType
	return instrument
}

var idTypeMap map[finance.InstrumentIdentifier]api.MappingJobIdType = map[finance.InstrumentIdentifier]api.MappingJobIdType{
	finance.InstrumentIdentifierTicker: api.TICKER,
	finance.InstrumentIdentifierISIN:   api.IDISIN,
	finance.InstrumentIdentifierFIGI:   api.IDBBGLOBAL,
}

func instrumentToMappingJob(instrument *finance.Instrument) *api.MappingJob {
	hasFIGI := instrument.HasIdentifier(finance.InstrumentIdentifierFIGI)
	hasTicker := instrument.HasIdentifier(finance.InstrumentIdentifierTicker)
	if hasFIGI && hasTicker {
		return nil
	}
	var instrumentIdentifer finance.InstrumentIdentifier
	var micCode *string
	if hasTicker {
		instrumentIdentifer = finance.InstrumentIdentifierTicker
		micCode = &instrument.MIC
	} else if hasFIGI {
		instrumentIdentifer = finance.InstrumentIdentifierFIGI
	} else if instrument.HasIdentifier(finance.InstrumentIdentifierISIN) {
		instrumentIdentifer = finance.InstrumentIdentifierISIN
		micCode = &instrument.MIC
	} else {
		return nil
	}
	idValue := api.MappingJob_IdValue{}
	identifier, _ := instrument.Identifier(instrumentIdentifer)
	idValue.FromMappingJobIdValue0(identifier)
	mappingJob := &api.MappingJob{
		IdType:  idTypeMap[instrumentIdentifer],
		IdValue: idValue,
		MicCode: micCode,
	}
	return mappingJob
}

func mappingJobResultToFigiResult(result *api.MappingJobResult) (*api.FigiResult, string) {
	figiList, err := result.AsMappingJobResultFigiList()
	if err != nil || figiList.Data == nil {
		notFound, _ := result.AsMappingJobResultFigiNotFound()
		return nil, ptrString(notFound.Warning)
	}
	figiResults := *figiList.Data
	if len(figiResults) != 1 {
		return nil, ""
	}
	return &figiResults[0], ""
}

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
