# Makefile Template
# Adjust variables and paths according to your specific project needs.

# Project Variables
PROJECT_NAME := gitops-nginx
BUILD_DIR := bin
CMD_DIR := cmd/gitops-nginx
UI_DIR := ui

# Tool Commands
GO := go
NPM := npm

# .PHONY tells Make that these targets are not actual files
.PHONY: all help build build-backend build-frontend run test clean

# Default target
all: build

# Help: Lists available commands
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build both backend and frontend"
	@echo "  build-backend   Build the Go backend binary"
	@echo "  build-frontend  Install dependencies and build the React frontend"
	@echo "  run             Run the backend application (Go)"
	@echo "  test            Run backend tests"
	@echo "  clean           Remove build artifacts"
	@echo "  help            Show this help message"

# Build complete project
build: build-backend build-frontend

# Build Backend (Go)
build-backend:
	@echo "==> Building Backend..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(PROJECT_NAME) $(CMD_DIR)/main.go
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
	$(GO) run $(CMD_DIR)/main.go

# Run Tests
test:
	@echo "==> Running Tests..."
	$(GO) test ./... -v

# Clean build artifacts
clean:
	@echo "==> Cleaning..."
	rm -rf $(BUILD_DIR)
	# Uncomment below to clean frontend artifacts as well
	# rm -rf $(UI_DIR)/node_modules $(UI_DIR)/dist
	@echo "Clean complete."
