## Go module for unified finance service provider access

[![GoDoc](https://godoc.org/github.com/tdrn-org/go-finance?status.svg)](https://godoc.org/github.com/tdrn-org/go-finance)
[![Build](https://github.com/tdrn-org/go-finance/actions/workflows/build.yml/badge.svg)](https://github.com/tdrn-org/go-finance/actions/workflows/build.yml)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=tdrn-org_go-finance&metric=coverage)](https://sonarcloud.io/summary/new_code?id=tdrn-org_go-finance)

### Provided services

The go-finance module provides the following services via different providers:
* FX service: Query currency exchange rates.
* SymbolResolver service: Search Symbols
* Equity service: Query Quotes

The following providers are available:
| Provider | FX | SymbolResolver | Equity | Comment |
|----------|----|----------------|--------|---------|
|[Alpha Vantage](https://www.alphavantage.co)|&check;|&check;|&check;|API key required|
|[Consorsbank TAPI](https://www.consorsbank.de/web/Wertpapierhandel/Trading-Software/ActiveTrader#API)|&check;|&check;|&check;|Requires Consorsbank depot and Active Trader|
|[Frankfurter](https://frankfurter.dev/)|&check;|||Free|
|[OpenFIGI](https://www.openfigi.com/api/overview)||&check;||API key optional|
|[Twelve Data](https://twelvedata.com/)|&check;|&check;|&check;|API key required|

### Development

The default goal runs tests and linters and never rewrites tracked files:

```sh
make check     # go test + go vet + staticcheck
make tidy      # go mod verify + go mod tidy
```

#### Generated code

The gRPC/protobuf stubs (`consorsbank/proto/*.pb.go`) and the OpenAPI-derived
sources under `openfigi/api/` are **tracked in this repository**, because
consumers import them. They are regenerated only on demand:

```sh
make generate  # requires protoc on PATH; needs no API keys
```

**Rule:** whenever a `.proto` file or an OpenAPI spec changes, run `make generate`
and commit the regenerated sources in the same commit.

`make check` deliberately does **not** generate. Regeneration is machine-specific:
protoc stamps its own version into a comment header of every generated file, so
running it as part of `check` turns every build into a working-tree change that
has nothing to do with the code being reviewed.

The code generators themselves are pinned through the `tool` directives in
`go.mod` (`protoc-gen-go`, `protoc-gen-go-grpc`, `oapi-codegen`, `staticcheck`),
so all machines and CI produce identical output. `protoc` is intentionally **not**
pinned — it only contributes a version comment, and mandating one protoc version
across developer machines would buy nothing.

### License

This project is subject to the the Apache License, Version 2.0. See LICENSE information for details.

### Important Disclaimer

This project is an independent open-source software library for accessing
financial data from third-party providers.

#### Provider Terms

Each provider may impose specific licensing, redistribution, attribution,
commercial use, and rate-limit requirements.

Users are responsible for reviewing and complying with the terms of the
providers they choose to use.

#### No Financial Advice

This software and any data obtained through it are provided solely for
informational and technical purposes. Nothing in this project constitutes
investment advice, financial advice, tax advice, legal advice, or a
recommendation to buy, sell, or hold any financial instrument.

#### Third-Party Data

All market data, exchange rates, stock prices, and related information are
provided by external services. The maintainers of this project do not create,
verify, audit, or guarantee the accuracy, completeness, timeliness, or
availability of such data.

#### No Warranty

This project is provided "AS IS", without warranties or conditions of any kind,
express or implied, including but not limited to warranties of accuracy,
reliability, merchantability, fitness for a particular purpose, or
non-infringement.

#### User Responsibility

Users are solely responsible for:

* Verifying the correctness of any data before relying on it.
* Complying with the terms of service and licensing conditions of any
  underlying data provider.
* Evaluating whether the software is suitable for their intended use.

#### Not for Mission-Critical Financial Decisions

This software is not intended for use as the sole basis for investment
decisions, automated trading systems, regulatory reporting, risk management,
or any other activity where inaccurate, delayed, or unavailable data could
result in financial loss or legal liability.

#### No Affiliation

This project is not affiliated with, endorsed by, sponsored by, or otherwise
associated with any financial data provider unless explicitly stated.