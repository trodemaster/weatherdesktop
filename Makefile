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

# Rebuild everything (Docker + host binary)
.PHONY: rebuild
rebuild:
	@echo "Cleaning cached assets..."
	@rm -f assets/*.png assets/*.jpg assets/*.html
	@echo "✓ Cached assets cleaned"
	@$(MAKE) docker-build build docker-restart
	@echo "✓ Complete rebuild finished"
	@echo "  - Docker image: weatherdesktop-wd-worker"
	@echo "  - Host binary: $(BUILD_DIR)/$(BINARY_NAME)"

# Build the binary for current OS/arch
.PHONY: build
build:
	@echo "Building for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wd
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
# Note: Does NOT delete rendered/*.jpg files (they are archived images)
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -rf assets/*.png assets/*.jpg assets/*.html
	@echo "✓ Cleaned (rendered images preserved)"

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker compose build
	@echo "✓ Docker image built"

.PHONY: docker-up
docker-up:
	@echo "Starting wd-worker container..."
	docker compose up -d --wait
	@echo "✓ Container started and healthy"

.PHONY: docker-down
docker-down:
	@echo "Stopping wd-worker container..."
	docker compose down
	@echo "✓ Container stopped"

.PHONY: docker-restart
docker-restart:
	@echo "Restarting wd-worker container..."
	docker compose restart wd-worker
	@echo "✓ Container restarted"

.PHONY: docker-logs
docker-logs:
	docker compose logs -f wd-worker

.PHONY: docker-shell
docker-shell:
	docker compose exec wd-worker sh

.PHONY: docker-ps
docker-ps:
	docker compose ps

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
	@echo "Build targets:"
	@echo "  make rebuild       - Rebuild everything (Docker + host binary)"
	@echo "  make build         - Build host binary (requires CGO)"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make deps          - Install/update Go dependencies"
	@echo ""
	@echo "Docker targets:"
	@echo "  make docker-build  - Build Docker image with Playwright"
	@echo "  make docker-up     - Start wd-worker container"
	@echo "  make docker-down   - Stop wd-worker container"
	@echo "  make docker-restart- Restart wd-worker container"
	@echo "  make docker-logs   - Follow container logs"
	@echo "  make docker-shell  - Open shell in container"
	@echo "  make docker-ps     - Show container status"
	@echo ""
	@echo "Run targets:"
	@echo "  make run           - Run full pipeline (scrape, download, crop, render, set desktop)"
	@echo "  make run-debug     - Run with debug output"
	@echo "  make list-targets  - List all scrape targets"
	@echo ""
	@echo "Development:"
	@echo "  make fmt           - Format Go code"
	@echo "  make lint          - Run linter"
	@echo "  make test          - Run tests"
	@echo ""
	@echo "Current configuration:"
	@echo "  OS:   $(GOOS)"
	@echo "  Arch: $(GOARCH)"

