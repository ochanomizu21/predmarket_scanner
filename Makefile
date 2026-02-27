.PHONY: build clean run test

BINARY_NAME=predmarket-scanner
BUILD_DIR=bin

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	go clean

run:
	@go run cmd/main.go --help

test:
	@go test ./...

deps:
	@go mod tidy
	@go mod download
