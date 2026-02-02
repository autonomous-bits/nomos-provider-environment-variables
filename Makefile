.PHONY: build test lint clean install-tools cross-compile

# Version injection
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Build targets
build:
	go build $(LDFLAGS) -o nomos-provider-environment-variables ./cmd/provider

test:
	go test -v -race -cover ./...

test-integration:
	go test -v -race -tags=integration ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -f nomos-provider-environment-variables
	rm -f coverage.out coverage.html
	rm -rf dist/

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Cross-compilation targets
cross-compile: clean
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/nomos-provider-environment-variables-$(VERSION)-darwin-amd64 ./cmd/provider
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/nomos-provider-environment-variables-$(VERSION)-darwin-arm64 ./cmd/provider
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/nomos-provider-environment-variables-$(VERSION)-linux-amd64 ./cmd/provider
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/nomos-provider-environment-variables-$(VERSION)-linux-arm64 ./cmd/provider
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/nomos-provider-environment-variables-$(VERSION)-windows-amd64.exe ./cmd/provider
	cd dist && shasum -a 256 * > SHA256SUMS

# Development workflow
dev: lint test build
