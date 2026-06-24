GOLANGCI_LINT_VERSION := v2.12.2
PACKAGES := ./cmd/... ./internal/... ./pkg/...

.PHONY: test check build

test:
	go test $(PACKAGES)

check:
	go vet $(PACKAGES)
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run $(PACKAGES)
	go run golang.org/x/vuln/cmd/govulncheck@latest $(PACKAGES)

build:
	go build -o bin/progames ./cmd/progames/
