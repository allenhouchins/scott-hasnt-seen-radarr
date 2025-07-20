# Scott Hasn't Seen Radarr List Generator Makefile

.PHONY: build test run clean install

# Binary name
BINARY_NAME=scott-hasnt-seen-radarr

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go
	@echo "Build complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	go run main.go

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod download

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f scott_hasnt_seen_radarr*.json

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe main.go
	@echo "Multi-platform build complete!"

# Run with test data (for development)
dev:
	@echo "Running in development mode..."
	TMDB_API_KEY=dummy_key go run main.go

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Show help
help:
	@echo "Available commands:"
	@echo "  build      - Build the application"
	@echo "  test       - Run tests"
	@echo "  run        - Run the application"
	@echo "  install    - Install dependencies"
	@echo "  clean      - Clean build artifacts"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  dev        - Run in development mode"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  help       - Show this help" 