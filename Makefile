MAKEFLAGS += --no-print-directory

GOBIN ?= $(shell go env GOPATH)/bin

export PATH := $(GOBIN):$(PATH)

.DEFAULT_GOAL := check

# Generated code (*.pb.go, OpenAPI-derived files) is committed to this repository.
# Two rules keep that deterministic across machines and CI:
#
#  1. 'check' never generates. Regenerating rewrites tracked files with the local
#     toolchain's version stamp — a machine-specific working-tree change that has
#     nothing to do with the code under review. 'make generate' is explicit.
#  2. The code generators are pinned through the 'tool' directives in go.mod, so
#     every machine and CI produce identical output. protoc itself is deliberately
#     NOT pinned: it only contributes a version comment.

.PHONE: deps
deps:
	go mod download -x

.PHONE: prototools
prototools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONE: linttools
linttools:
	go install honnef.co/go/tools/cmd/staticcheck

.PHONE: generate
generate: deps prototools
	@echo "==> protoc        $$(protoc --version)"
	@echo "==> protoc-gen-go $$(protoc-gen-go --version)"
	@echo "==> Generated files are tracked: commit the regenerated sources."
	go generate ./...

.PHONE: tidy
tidy:
	go mod verify
	go mod tidy

.PHONE: vet
vet:
	go vet ./...

.PHONE: staticcheck
staticcheck: linttools
	$(GOBIN)/staticcheck ./...

.PHONE: lint
lint: vet staticcheck

.PHONE: test
test:
	go test -v -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...

.PHONE: check
check: test lint

.PHONE: clean
clean:
	go clean ./...
