# Makefile for Telegram Chat Summarizer Bot

.PHONY: all build clean test run help

# Variables
BINARY_NAME=telegram-summarizer
BIN_DIR=bin
CMD_DIR=cmd
BUILD_FLAGS=-ldflags="-s -w"

# Default target
all: clean build

# Build the unified binary
build:
	# @echo "🔨 Building $(BINARY_NAME)..."
	# @mkdir -p $(BIN_DIR)
	# @go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	# @echo "✅ Build complete: $(BIN_DIR)/$(BINARY_NAME)"
	# @ls -lh $(BIN_DIR)/$(BINARY_NAME)
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@GOMAXPROCS=1 CGO_ENABLED=1 go build -p 1 $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "✅ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Build old binaries (deprecated, for backup)
build-old:
	@echo "🔨 Building old binaries..."
	@mkdir -p $(BIN_DIR)
	@go build $(BUILD_FLAGS) -o $(BIN_DIR)/bot $(CMD_DIR)/bot/main.go
	@go build $(BUILD_FLAGS) -o $(BIN_DIR)/scraper $(CMD_DIR)/scraper/main.go
	@echo "✅ Old binaries built"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -f $(BIN_DIR)/$(BINARY_NAME)
	@rm -f $(BIN_DIR)/bot $(BIN_DIR)/scraper
	@rm -f *.log
	@echo "✅ Clean complete"

# Run the bot (all mode)
run:
	@echo "🚀 Starting $(BINARY_NAME)..."
	@$(BIN_DIR)/$(BINARY_NAME) --phone $(PHONE)

# Run bot only
run-bot:
	@echo "🤖 Starting bot only..."
	@$(BIN_DIR)/$(BINARY_NAME) --mode bot

# Run scraper only
run-scraper:
	@echo "📱 Starting scraper only..."
	@$(BIN_DIR)/$(BINARY_NAME) --mode scraper --phone $(PHONE)

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Format code
fmt:
	@echo "📝 Formatting code..."
	@go fmt ./...
	@echo "✅ Format complete"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@golangci-lint run || echo "Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

# Show version
version:
	@$(BIN_DIR)/$(BINARY_NAME) -version

# Help
help:
	@echo "Telegram Chat Summarizer Bot - Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build the unified binary"
	@echo "  make build-old      - Build old separate binaries (deprecated)"
	@echo "  make clean          - Remove build artifacts and logs"
	@echo "  make run            - Run bot and scraper (requires PHONE env)"
	@echo "  make run-bot        - Run bot only"
	@echo "  make run-scraper    - Run scraper only (requires PHONE env)"
	@echo "  make test           - Run tests"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make deps           - Install dependencies"
	@echo "  make version        - Show version"
	@echo "  make help           - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  PHONE               - Phone number for scraper (e.g., +628123456789)"
	@echo "  TELEGRAM_TOKEN      - Telegram bot token"
	@echo "  GEMINI_API_KEY      - Google Gemini API key"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  PHONE=+628123456789 make run"
	@echo "  make run-bot"
