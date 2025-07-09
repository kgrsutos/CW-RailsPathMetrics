.PHONY: build test lint clean run deps test-coverage install-tools help

BINARY_NAME=cwrstats
MAIN_PATH=./cmd/cwrstats
GO=go
GOLANGCI_LINT=golangci-lint

build:
	$(GO) build -o $(BINARY_NAME) $(MAIN_PATH)

test:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

lint:
	$(GOLANGCI_LINT) run

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

run: build
	./$(BINARY_NAME)

deps:
	$(GO) mod download
	$(GO) mod tidy


install-tools:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0

help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  test           - Run tests with coverage"
	@echo "  test-coverage  - Run tests and display coverage"
	@echo "  lint           - Run linters"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the binary"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  install-tools  - Install development tools"