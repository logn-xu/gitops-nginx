# Project Variables
PROJECT_NAME := gitops-nginx
BUILD_DIR := bin
CMD_DIR := cmd/gitops-nginx
UI_DIR := ui
EMBED_DIR := $(CMD_DIR)/dist

# Tool Commands
GO := go
NPM := npm

# Build platforms
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

# .PHONY tells Make that these targets are not actual files
.PHONY: all help build build-embed build-all release-all run test clean

# Default target
all: build-embed

# Help: Lists available commands
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build backend only"
	@echo "  build-embed     Build with embedded frontend (default)"
	@echo "  build-all       Build for all platforms (embedded)"
	@echo "  release-all     Build and package all platforms (.tar.gz)"
	@echo "  run             Run the application"
	@echo "  test            Run tests"
	@echo "  clean           Remove build artifacts"
	@echo "  help            Show this help message"

# Build backend only
build:
	@echo "==> Building Backend..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(PROJECT_NAME) ./$(CMD_DIR)

# Build with embedded frontend (single binary)
build-embed: build-frontend
	@echo "==> Preparing embedded assets..."
	@rm -rf $(EMBED_DIR)
	@mkdir -p $(EMBED_DIR)
	@if [ -d "$(UI_DIR)/dist" ]; then cp -r $(UI_DIR)/dist/* $(EMBED_DIR)/; fi
	@echo "==> Building Backend (Embedded)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(PROJECT_NAME) ./$(CMD_DIR)

# Build for all platforms
build-all: build-frontend
	@echo "==> Preparing embedded assets for cross-compilation..."
	@rm -rf $(EMBED_DIR)
	@mkdir -p $(EMBED_DIR)
	@if [ -d "$(UI_DIR)/dist" ]; then cp -r $(UI_DIR)/dist/* $(EMBED_DIR)/; fi
	@echo "==> Building for all platforms..."
	@$(foreach PLATFORM,$(PLATFORMS), \
		GOOS=$(word 1,$(subst /, ,$(PLATFORM))) \
		GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
		CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(PROJECT_NAME)-$(word 1,$(subst /, ,$(PLATFORM)))-$(word 2,$(subst /, ,$(PLATFORM))) ./$(CMD_DIR); \
	)

# Build and package all platforms
release-all: build-all
	@echo "==> Packaging all platforms..."
	@mkdir -p $(BUILD_DIR)/release
	@$(foreach PLATFORM,$(PLATFORMS), \
		OS=$(word 1,$(subst /, ,$(PLATFORM))); \
		ARCH=$(word 2,$(subst /, ,$(PLATFORM))); \
		tar -C $(BUILD_DIR) -czf $(BUILD_DIR)/release/$(PROJECT_NAME)-$$OS-$$ARCH.tar.gz $(PROJECT_NAME)-$$OS-$$ARCH; \
	)

# Internal target: Build Frontend
build-frontend:
	@echo "==> Building Frontend..."
	cd $(UI_DIR) && $(NPM) install && $(NPM) run build

# Run Backend (Development)
run:
	@echo "==> Running Backend..."
	$(GO) run ./$(CMD_DIR)

# Run Tests
test:
	@echo "==> Running Tests..."
	$(GO) test ./... -v

# Clean build artifacts
clean:
	@echo "==> Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(EMBED_DIR)
	@echo "Clean complete."