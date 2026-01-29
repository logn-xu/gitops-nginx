# Makefile Template
# Adjust variables and paths according to your specific project needs.

# Project Variables
PROJECT_NAME := gitops-nginx
BUILD_DIR := bin
CMD_DIR := cmd/gitops-nginx
UI_DIR := ui
EMBED_DIR := $(CMD_DIR)/dist

# Tool Commands
GO := go
NPM := npm

# .PHONY tells Make that these targets are not actual files
.PHONY: all help build build-embed build-backend build-frontend run test clean

# Default target
all: build

# Help: Lists available commands
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build backend only (no embedded frontend)"
	@echo "  build-embed     Build with embedded frontend (single binary)"
	@echo "  build-backend   Build the Go backend binary"
	@echo "  build-frontend  Install dependencies and build the React frontend"
	@echo "  run             Run the backend application (Go)"
	@echo "  test            Run backend tests"
	@echo "  clean           Remove build artifacts"
	@echo "  help            Show this help message"

# Build backend only
build: build-backend

# Build with embedded frontend (single binary)
build-embed: build-frontend embed-frontend build-backend
	@echo "==> Embedded build complete: $(BUILD_DIR)/$(PROJECT_NAME)"

# Copy frontend dist to cmd directory for embedding
embed-frontend:
	@echo "==> Copying frontend dist for embedding..."
	@rm -rf $(EMBED_DIR)
	@mkdir -p $(EMBED_DIR)
	@cp -r $(UI_DIR)/dist/* $(EMBED_DIR)/
	@echo "Frontend copied to $(EMBED_DIR)"

# Build Backend (Go)
build-backend:
	@echo "==> Building Backend..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(PROJECT_NAME) ./$(CMD_DIR)
	@echo "Backend binary created at $(BUILD_DIR)/$(PROJECT_NAME)"

# Build Frontend (Node/React)
build-frontend:
	@echo "==> Building Frontend..."
	cd $(UI_DIR) && $(NPM) install
	cd $(UI_DIR) && $(NPM) run build
	@echo "Frontend build complete."

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
