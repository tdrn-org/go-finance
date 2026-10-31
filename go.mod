module github.com/tdrn-org/go-finance

go 1.27.0

toolchain go1.27.1

require (
	github.com/oapi-codegen/runtime v1.8.0
	github.com/stretchr/testify v1.12.1
	github.com/tdrn-org/go-cache v0.1.9
	github.com/tdrn-org/go-config-toml v0.1.1
	github.com/twelvedata/twelvedata-go v1.0.14
	go.opentelemetry.io/otel v1.47.0
	go.opentelemetry.io/otel/trace v1.47.0
	google.golang.org/grpc v1.84.0
	google.golang.org/protobuf v1.36.12
	gopkg.in/dnaeon/go-vcr.v4 v4.0.7
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dprotaso/go-yit v0.0.0-20261016225440-96fdf4ef124e // indirect
	github.com/getkin/kin-openapi v0.152.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v1.0.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/maypok86/otter/v2 v2.4.0 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.8.1 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/redis/go-redis/v9 v9.23.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/speakeasy-api/jsonpath v0.7.0 // indirect
	github.com/speakeasy-api/openapi v1.26.0 // indirect
	github.com/speakeasy-api/yaml-jsonpath v0.0.0-20261016002013-07e5f48e6d17 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.6 // indirect
	golang.org/x/exp/typeparams v0.0.0-20261014143641-f0251eb4aa3e // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261019180522-ce908d29299b // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.8.2 // indirect
)

tool (
	github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
	honnef.co/go/tools/cmd/staticcheck
)
