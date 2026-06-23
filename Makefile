# Window specific variables
ifeq ($(OS),Windows_NT)
	EXE := .exe
else
	EXE :=
endif

GO := go
BIN := bin/progames$(EXE)
GOLANGCI_LINT_VERSION := v2.12.2
GOVULNCHECK_VERSION := latest

.PHONY: tidy fmt test lint vuln fix check build

tidy:
	${GO} mod tidy

fmt: 
	${GO} fmt ./...

test:
	${GO} test $(shell ${GO} list ./... | grep -v '/artifacts/')

lint:
	${GO} run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} run

vuln:
	${GO} run golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} ./...

fix: fmt tidy

check: test lint vuln

build:
	$(GO) build -o $(BIN) cmd/progames/
