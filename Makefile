SHELL := /bin/zsh

COVERAGE_FILE := coverage.out

.PHONY: help run test lint vet coverage coverage-html security

help:
	@printf "Targets:\n"
	@printf "  make run       Start clawmem locally\n"
	@printf "  make test      Run unit tests\n"
	@printf "  make lint      Run formatting check, vet, and golangci-lint\n"
	@printf "  make vet       Run go vet\n"
	@printf "  make coverage  Run tests with coverage output\n"
	@printf "  make security  Run gosec and govulncheck when installed\n"

run:
	go run ./cmd/clawmem

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
