# ──────────────────────────────────────────────────────────────────────────────
#  go-task-offloader  —  Developer Makefile
# ──────────────────────────────────────────────────────────────────────────────

BINARY      := go-task-offloader
CMD         := ./cmd/worker
BUILD_DIR   := ./bin
GO          := go
GOFLAGS     := -ldflags="-s -w"

# Detect OS for platform-specific commands
UNAME := $(shell uname -s)

.DEFAULT_GOAL := help

# ── Help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ── Build ─────────────────────────────────────────────────────────────────────
.PHONY: build
build: ## Compile the binary into ./bin/
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)
	@echo "✓ Binary: $(BUILD_DIR)/$(BINARY)"

.PHONY: build-cgo
build-cgo: ## Compile with CGO enabled (required for govips/libvips)
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)
	@echo "✓ Binary (CGO): $(BUILD_DIR)/$(BINARY)"

# ── Run ───────────────────────────────────────────────────────────────────────
.PHONY: run
run: ## Run the worker (reads config from env / .env)
	$(GO) run $(CMD)

# ── Test ──────────────────────────────────────────────────────────────────────
.PHONY: test
test: ## Run all unit tests
	$(GO) test -v -race -count=1 ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# ── Code quality ──────────────────────────────────────────────────────────────
.PHONY: lint
lint: ## Run golangci-lint (install: https://golangci-lint.run/usage/install/)
	golangci-lint run ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format all Go source files
	$(GO) fmt ./...

.PHONY: tidy
tidy: ## Tidy and verify go.mod / go.sum
	$(GO) mod tidy
	$(GO) mod verify

# ── Docker ────────────────────────────────────────────────────────────────────
.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t $(BINARY):latest .

.PHONY: docker-up
docker-up: ## Start the full Docker Compose stack (worker + Redis)
	docker compose up --build

.PHONY: docker-down
docker-down: ## Stop and remove Docker Compose containers
	docker compose down

.PHONY: docker-logs
docker-logs: ## Tail worker logs
	docker compose logs -f worker

# ── Utilities ─────────────────────────────────────────────────────────────────
.PHONY: clean
clean: ## Remove build artefacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html

.PHONY: health
health: ## Ping the running worker's health endpoint
	curl -s http://localhost:8080/health | python3 -m json.tool

.PHONY: metrics
metrics: ## Fetch the running worker's metrics
	curl -s http://localhost:8080/metrics | python3 -m json.tool

.PHONY: push-test-job
push-test-job: ## Push a sample resize job to Redis for manual testing
	redis-cli RPUSH queues:default '{"uuid":"test-001","displayName":"App\\\\Jobs\\\\ProcessImageJob","maxTries":3,"timeout":60,"attempts":0,"id":"test-001","data":{"source_url":"/tmp/test.jpg","output_url":"/tmp/out.jpg","action":"resize","width":800,"height":600}}'
	@echo "✓ Test job pushed to queues:default"
