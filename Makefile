.PHONY: build run test test-race test-coverage lint tidy clean deps help

BINARY  := opencode-sm
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
BUILD   := ./bin/$(BINARY)
PKGS    := ./...

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BUILD) ./cmd/opencode-sm

run: build
	$(BUILD)

watch-run: build
	$(BUILD) --watch

test:
	go test $(PKGS)

test-race:
	go test -race $(PKGS)

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic $(PKGS)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run $(PKGS)

tidy:
	go mod tidy

deps:
	go get -u $(PKGS)

clean:
	rm -rf bin coverage.out coverage.html

help:
	@echo "Available targets:"
	@echo "  build           Build the binary to ./bin/$(BINARY)"
	@echo "  run             Build and run the binary"
	@echo "  watch-run       Build and run with watch mode enabled"
	@echo "  test            Run unit tests"
	@echo "  test-race       Run tests with race detector"
	@echo "  test-coverage   Run tests and produce coverage.html"
	@echo "  lint            Run golangci-lint"
	@echo "  tidy            Run go mod tidy"
	@echo "  deps            Update dependencies"
	@echo "  clean           Remove build artifacts"
