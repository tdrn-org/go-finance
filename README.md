## Go module for Finance service provider access
[![GoDoc](https://godoc.org/github.com/tdrn-org/go-finance?status.svg)](https://godoc.org/github.com/tdrn-org/go-finance)
[![Build](https://github.com/tdrn-org/go-finance/actions/workflows/build.yml/badge.svg)](https://github.com/tdrn-org/go-finance/actions/workflows/build.yml)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=tdrn-org_go-finance&metric=coverage)](https://sonarcloud.io/summary/new_code?id=tdrn-org_go-finance)

### Supported services
The go-finance module wraps the following services:
* FX service: Query currency exchange rates.
* SymbolResolver service: Search Symbols
* Equity service: Query Quotes

The following providers are integrated:
| Provider | FX | SymbolResolver | Equity | Comment |
|----------|----|----------------|--------|---------|
|[Alpha Vantage](https://www.alphavantage.co)|&check;|&check;|&check;|API key required|
|[Consorsbank TAPI](https://www.consorsbank.de/web/Wertpapierhandel/Trading-Software/ActiveTrader#API)|&check;|&check;|&check;|Requires Consorsbank depot and Active Trader|
|[Frankfurter](https://frankfurter.dev/)|&check;|||Free|
|[OpenFIGI](https://www.openfigi.com/api/overview)||&check;||API key optional|
|[Twelve Data](https://twelvedata.com/)|&check;|&check;|&check;|API key required|

### License
This project is subject to the the Apache License, Version 2.0. See LICENSE information for details.