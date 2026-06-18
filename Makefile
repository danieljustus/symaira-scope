BINARY := symscope
PKG := ./cmd/symscope

.PHONY: build test vet lint run serve clean

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

run: build
	./$(BINARY) scan

serve: build
	./$(BINARY) serve

clean:
	rm -f $(BINARY)
	rm -rf dist
