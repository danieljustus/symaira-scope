BINARY := symscope
PKG := ./cmd/symscope
GOLANGCI_LINT ?= golangci-lint
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)

.PHONY: build test vet lint run serve install clean

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { echo "$(GOLANGCI_LINT) is required for linting" >&2; exit 1; }
	$(GOLANGCI_LINT) run --timeout=5m

run: build
	./$(BINARY) scan

serve: build
	./$(BINARY) serve

install:
	CGO_ENABLED=0 go install -ldflags "-X github.com/danieljustus/symaira-scope/internal/version.Version=$(VERSION)" $(PKG)

clean:
	rm -f $(BINARY)
	rm -rf dist
