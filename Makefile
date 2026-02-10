BINARY := termdash
BUILD_DIR := bin
DIST_DIR := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOMOD := $(GOCMD) mod

.PHONY: all build run test clean deps lint \
        build-linux build-darwin build-windows build-all \
        docker-build docker-image docker-run \
        install uninstall release

# Default target
all: build

# Build for current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/termdash

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY)

# Run tests
test:
	$(GOTEST) ./... -v

# Run tests with coverage
test-coverage:
	$(GOTEST) ./... -v -coverprofile=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	$(GOCLEAN)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# ============================================================================
# Cross-compilation targets
# ============================================================================

# Build for Linux (amd64)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY)-linux-amd64 ./cmd/termdash

# Build for Linux ARM64
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY)-linux-arm64 ./cmd/termdash

# Build for macOS (amd64) — CGO required for gopsutil on darwin
build-darwin:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY)-darwin-amd64 ./cmd/termdash

# Build for macOS ARM64 (Apple Silicon) — CGO required for gopsutil on darwin
build-darwin-arm64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY)-darwin-arm64 ./cmd/termdash

# Build for Windows
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY)-windows-amd64.exe ./cmd/termdash

# Build for all platforms
build-all: clean
	@mkdir -p $(DIST_DIR)
	@echo "Building for all platforms..."
	@./scripts/build-all.sh $(VERSION)

# ============================================================================
# Docker targets
# ============================================================================

# Build Docker image
docker-image:
	docker build -t $(BINARY):$(VERSION) -t $(BINARY):latest .

# Build using Docker (no local Go required)
docker-build:
	@./scripts/docker-build.sh linux amd64 $(VERSION)

# Build for specific platform using Docker
docker-build-linux-arm64:
	@./scripts/docker-build.sh linux arm64 $(VERSION)

docker-build-darwin:
	@./scripts/docker-build.sh darwin amd64 $(VERSION)

docker-build-darwin-arm64:
	@./scripts/docker-build.sh darwin arm64 $(VERSION)

# Run in Docker container (for testing)
docker-run: docker-image
	docker run -it --rm \
		--pid=host \
		--privileged \
		$(BINARY):latest

# ============================================================================
# Installation targets
# ============================================================================

# Install to /usr/local/bin
install: build
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

# Uninstall from /usr/local/bin
uninstall:
	rm -f /usr/local/bin/$(BINARY)

# ============================================================================
# Release target
# ============================================================================

release: clean build-all
	@echo "Creating release $(VERSION)..."
	@cd $(DIST_DIR) && \
		for file in $(BINARY)-*; do \
			if [ -f "$$file" ]; then \
				tar -czf "$$file.tar.gz" "$$file" 2>/dev/null || true; \
				zip "$$file.zip" "$$file" 2>/dev/null || true; \
			fi \
		done
	@echo "Release artifacts created in $(DIST_DIR)/"

# ============================================================================
# Help
# ============================================================================

help:
	@echo "termdash - Terminal-based system monitor"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build              Build for current platform"
	@echo "  run                Build and run"
	@echo "  test               Run tests"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  clean              Clean build artifacts"
	@echo "  deps               Download dependencies"
	@echo "  lint               Run linter"
	@echo ""
	@echo "Cross-compilation:"
	@echo "  build-linux        Build for Linux amd64"
	@echo "  build-linux-arm64  Build for Linux arm64"
	@echo "  build-darwin       Build for macOS amd64"
	@echo "  build-darwin-arm64 Build for macOS arm64 (Apple Silicon)"
	@echo "  build-windows      Build for Windows amd64"
	@echo "  build-all          Build for all platforms"
	@echo ""
	@echo "Docker:"
	@echo "  docker-image       Build Docker image"
	@echo "  docker-build       Build binary using Docker"
	@echo "  docker-run         Run in Docker container"
	@echo ""
	@echo "Installation:"
	@echo "  install            Install to /usr/local/bin"
	@echo "  uninstall          Remove from /usr/local/bin"
	@echo ""
	@echo "Release:"
	@echo "  release            Build all platforms and create archives"
