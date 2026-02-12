.PHONY: build run clean test deps help

# Build variables
BIN_DIR := bin
BINARY := $(BIN_DIR)/cron-engine-serve
GO := go
GOFLAGS := -q

# Build the cron engine binary
build:
	@echo "Building cron engine..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(GOFLAGS) -o $(BINARY) ./cmd/serve
	@echo "Built $(BINARY)"

# Run the cron engine
run: build
	@./$(BINARY)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@$(GO) clean $(GOFLAGS)

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test $(GOFLAGS) ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GO) mod tidy
	@$(GO) mod download

# Display help
help:
	@echo "Cron Engine UMC - Lifecycle management"
	@echo "========================================"
	@echo ""
	@echo "Available targets:"
	@echo "  make build      - Build the cron engine binary"
	@echo "  make run        - Build and run the cron engine"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make test       - Run unit tests"
	@echo "  make deps       - Download and tidy dependencies"
	@echo "  make help       - Display this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  KERNEL_SOCKET   - Path to kernel syscall socket (default: /tmp/ua_kernel.sock)"
	@echo "  LOG_LEVEL       - Logging level (default: info)"
