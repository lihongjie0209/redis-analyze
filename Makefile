.PHONY: build run test clean lint help release release-static list

BINARY    = redis-analyze
BUILD_DIR = build
VERSION   = $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS   = -ldflags="-s -w -X main.version=$(VERSION)"

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary for current platform
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

run: ## Build and run (pass ARGS for extra flags, e.g. make run ARGS="-P user:* -f json")
	go run . $(ARGS)

test: ## Run tests
	go test ./... -v -count=1

test-race: ## Run tests with race detector
	go test ./... -race -v -count=1

lint: ## Run golangci-lint
	golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed, skipping"

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	go clean

install: ## Install binary to $$GOPATH/bin
	go install $(LDFLAGS) .

# ─── Release builds (static, musl-compatible) ───────────────────────────────
#
# All builds use CGO_ENABLED=0 to produce fully static binaries.
# They work on any Linux distro (glibc, musl/alpine, busybox) without
# runtime library dependencies.

release: clean ## Build for all platforms (static, musl-compatible)
	@$(MAKE) release-linux
	@$(MAKE) release-darwin
	@$(MAKE) release-windows
	@echo ""
	@ls -lh $(BUILD_DIR)/*

release-linux: ## Linux x86_64 + ARM64 (fully static)
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY)-linux-arm64 .

release-darwin: ## macOS x86_64 + ARM64 (Apple Silicon)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .

release-windows: ## Windows amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe .

release-musl: CGO_ENABLED=0 ## Alias: same as release (CGO_ENABLED=0 = fully static)
release-musl: release

list: ## Show built binaries with type info
	@file $(BUILD_DIR)/$(BINARY)-* 2>/dev/null || echo "No builds found, run 'make release' first"

# ─── Quick test ─────────────────────────────────────────────────────────────

test-redis: build ## Test against local Redis (HOST/PORT env, default 127.0.0.1:6379)
	@echo "Testing against Redis at $(or $(HOST),127.0.0.1):$(or $(PORT),6379)..."
	$(BUILD_DIR)/$(BINARY) --host $(or $(HOST),127.0.0.1) --port $(or $(PORT),6379) \
		--prefix '*' --top 10

test-bench: build ## Benchmark: 10k keys, all scan modes
	@echo "=== Pipeline (default) ==="
	$(BUILD_DIR)/$(BINARY) --host $(or $(HOST),127.0.0.1) --port $(or $(PORT),6379) \
		--prefix '*' --top 3 --no-progress --timeout 60
	@echo ""
	@echo "=== Sequential ==="
	$(BUILD_DIR)/$(BINARY) --host $(or $(HOST),127.0.0.1) --port $(or $(PORT),6379) \
		--prefix '*' --top 3 --scan-mode sequential --no-progress --timeout 60
