MAKEFLAGS += --no-print-directory

GOBIN ?= $(shell go env GOPATH)/bin

.DEFAULT_GOAL := check

.PHONE: deps
deps:
	go mod download -x
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONE: testdeps
testdeps: deps
	go install honnef.co/go/tools/cmd/staticcheck@latest

.PHONE: generate
generate: deps
	go generate ./...

.PHONE: tidy
tidy:
	go mod verify
	go mod tidy

.PHONE: vet
vet: testdeps
	go vet ./...

.PHONE: staticcheck
staticcheck: testdeps
	$(GOBIN)/staticcheck ./...

.PHONE: lint
lint: vet staticcheck

.PHONE: test
test:
	go test -v -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...

.PHONE: check
check: generate test lint

.PHONE: clean
clean:
	go clean ./...
