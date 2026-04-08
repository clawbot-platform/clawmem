SHELL := /bin/zsh

COVERAGE_FILE := coverage.out
VERSION ?= $(shell (test -f VERSION && cat VERSION) || (git describe --tags --always --dirty 2>/dev/null || echo dev))
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'clawmem/internal/version.Version=$(VERSION)' \
           -X 'clawmem/internal/version.Commit=$(COMMIT)' \
           -X 'clawmem/internal/version.Date=$(BUILD_DATE)'

.PHONY: help run build test lint vet coverage coverage-html security

help:
	@printf "Targets:\n"
	@printf "  make run       Start clawmem locally with build metadata\n"
	@printf "  make build     Build clawmem binary with build metadata\n"
	@printf "  make test      Run unit tests\n"
	@printf "  make lint      Run formatting check, vet, and golangci-lint\n"
	@printf "  make vet       Run go vet\n"
	@printf "  make coverage  Run tests with coverage output\n"
	@printf "  make security  Run gosec and govulncheck when installed\n"

run:
	go run -ldflags "$(LDFLAGS)" ./cmd/clawmem

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/clawmem ./cmd/clawmem

test:
	go test ./...

lint: vet
	@test -z "$$(find cmd internal -name '*.go' -print | xargs gofmt -l)" || (echo "run gofmt -w on repository Go files" && exit 1)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed"

vet:
	go vet ./...

coverage:
	go test -covermode=atomic -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

coverage-html: coverage
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html

security:
	@command -v gosec >/dev/null 2>&1 && gosec ./... || echo "gosec not installed"
	@command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed"
