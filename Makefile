# Weather Desktop Makefile

# Detect OS and architecture from the host system
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Binary name
BINARY_NAME := wd

# Build directory
BUILD_DIR := .

# Go build flags
LDFLAGS := -s -w
BUILD_FLAGS := -ldflags="$(LDFLAGS)"

# Default target
.PHONY: all
all: build

# Build the binary for current OS/arch
.PHONY: build
build:
	@echo "Building for $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wd
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -rf assets/*.png assets/*.jpg assets/*.html
	@rm -rf rendered/*.jpg
	@echo "✓ Cleaned"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies installed"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

# Run linter (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install: brew install golangci-lint" && exit 1)
	golangci-lint run

# Show build info
.PHONY: info
info:
	@echo "Build Information:"
	@echo "  OS: $(GOOS)"
	@echo "  Architecture: $(GOARCH)"
	@echo "  Go Version: $(shell go version)"
	@echo "  Binary Name: $(BINARY_NAME)"
	@echo "  Output: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the full pipeline
.PHONY: run
run: build
	@echo "Running full pipeline..."
	./$(BINARY_NAME)

# Run with debug (does not set desktop wallpaper)
.PHONY: run-debug
run-debug: build
	@echo "Running with debug mode (desktop wallpaper will NOT be set)..."
	./$(BINARY_NAME) -debug

# List scrape targets
.PHONY: list-targets
list-targets: build
	@./$(BINARY_NAME) -list-targets

# Help target
.PHONY: help
help:
	@echo "Weather Desktop Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build        - Build binary for current OS/arch (default)"
	@echo "  make clean        - Remove build artifacts and generated files"
	@echo "  make deps         - Install/update Go dependencies"
	@echo "  make test         - Run tests"
	@echo "  make fmt          - Format Go code"
	@echo "  make lint         - Run linter (requires golangci-lint)"
	@echo "  make info         - Show build information"
	@echo "  make run          - Build and run full pipeline (production, uses lock)"
	@echo "  make run-debug    - Build and run with debug (test mode, no lock)"
	@echo "  make list-targets - Build and list scrape targets"
	@echo "  make help         - Show this help message"
	@echo ""
	@echo "Lock File Behavior:"
	@echo "  Production: Uses lock file ($$TMPDIR/wd.lock) to prevent conflicts"
	@echo "  Test mode:  Bypasses lock (safe to run alongside production)"
	@echo "    Triggers: -debug OR -scrape-target flags"
	@echo "    Effects:  Unique filenames, no desktop setting"
	@echo ""
	@echo "Current configuration:"
	@echo "  OS:   $(GOOS)"
	@echo "  Arch: $(GOARCH)"

