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

package proto

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go_opt=MAccessService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MAccountService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MCommon.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MDepotService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MOrderService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MSecurityService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MStockExchangeService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MTradingAPI.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go_opt=MTradingTypes.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-grpc_opt=MAccessService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MAccountService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MCommon.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MDepotService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MOrderService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MSecurityService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MStockExchangeService.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MTradingAPI.proto=github.com/tdrn-org/go-finance/consorsbank/proto --go-grpc_opt=MTradingTypes.proto=github.com/tdrn-org/go-finance/consorsbank/proto AccessService.proto AccountService.proto Common.proto DepotService.proto OrderService.proto SecurityService.proto StockExchangeService.proto TradingAPI.proto TradingTypes.proto
