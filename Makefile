SHELL := /bin/zsh

.PHONY: help run test lint vet security

help:
	@printf "Targets:\n"
	@printf "  make run       Start clawmem locally\n"
	@printf "  make test      Run unit tests\n"
	@printf "  make lint      Run formatting check and vet\n"
	@printf "  make vet       Run go vet\n"
	@printf "  make security  Run govulncheck if installed\n"

run:
	go run ./cmd/clawmem

test:
	go test ./...

lint: vet
	@test -z "$$(gofmt -l .)" || (echo "run gofmt -w ." && exit 1)

vet:
	go vet ./...

security:
	@command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed"
