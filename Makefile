BIN     := bin/ste
PKG     := ./cmd/ste
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Documents that must pass the tool's own rules.
DOCS := README.md CLAUDE.md docs

.PHONY: all build test check lint fmt vet tidy clean install

all: check lint

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# Fail if any file needs gofmt.
check: vet test
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

# Dogfood: this repository's own prose must pass.
lint: build
	$(BIN) lint --max-per100w 0.5 $(DOCS)

clean:
	rm -rf bin
