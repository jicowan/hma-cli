.PHONY: build test clean install lint

BINARY_NAME=hma-cli
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/hma-cli

# Build for Linux (for deployment to EKS nodes)
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/hma-cli
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/hma-cli

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-* coverage.out coverage.html

# Install to $GOPATH/bin
install:
	go install ./cmd/hma-cli

# Tidy dependencies
tidy:
	go mod tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# All checks before commit
check: fmt lint test
