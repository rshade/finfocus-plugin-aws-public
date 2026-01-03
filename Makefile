# All supported AWS regions (sourced from internal/pricing/regions.yaml)
# Uses Go-based YAML parser for robust parsing (replaces fragile awk/sed)
REGIONS_FILE := internal/pricing/regions.yaml
# Use a subshell for the directory change to keep it isolated (Issue #67)
PARSE_REGIONS := cd tools/parse-regions && go mod download -x 2>/dev/null && go run . -config ../../$(REGIONS_FILE) -format lines
REGIONS := $(shell $(PARSE_REGIONS) -field name)
REGIONS_CSV := $(shell $(PARSE_REGIONS) -field name | tr '\n' ',' | sed 's/,$$//')
REGION_COUNT := $(words $(REGIONS))

# Development version: latest tag + patch increment + "-dev" suffix
# Example: v0.0.12 -> 0.0.13-dev
LATEST_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LATEST_VERSION := $(shell echo $(LATEST_TAG) | sed 's/^v//')
MAJOR := $(shell echo $(LATEST_VERSION) | cut -d. -f1)
MINOR := $(shell echo $(LATEST_VERSION) | cut -d. -f2)
# Issue #178: Sanitize PATCH to default to 0 if non-numeric
PATCH := $(shell echo $(LATEST_VERSION) | cut -d. -f3 | grep -E '^[0-9]+$$' || echo "0")
NEXT_PATCH := $(shell echo $$(($(PATCH) + 1)))
DEV_VERSION := $(MAJOR).$(MINOR).$(NEXT_PATCH)-dev
LDFLAGS := -X main.version=$(DEV_VERSION)

.PHONY: all
all: build ## Default target - build the plugin

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

.PHONY: ensure
ensure: ## Install development dependencies (goreleaser, golangci-lint)
	@echo "Installing development dependencies..."
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed successfully"

.PHONY: develop
develop: ensure generate-pricing generate-carbon-data ## Setup development environment (install deps + generate data)
	@echo "Development environment ready"

.PHONY: generate-pricing
generate-pricing: ## Generate per-service pricing data for all regions
	@echo "Generating per-service pricing data for all regions..."
	@echo "Output: internal/pricing/data/{service}_{region}.json"
	@echo "Services: ec2, s3, rds, eks, lambda, dynamodb, elb, vpc"
	@go run ./tools/generate-pricing --regions $(REGIONS_CSV) --out-dir ./internal/pricing/data

.PHONY: generate-carbon-data
generate-carbon-data: ## Fetch CCF instance specs for carbon estimation
	@echo "Fetching Cloud Carbon Footprint instance specs..."
	@go run ./tools/generate-carbon-data --out-dir ./internal/carbon/data

.PHONY: generate-embeds
generate-embeds: ## Generate embed files from regions.yaml
	@echo "Generating embed files..."
	@$(MAKE) -C tools/generate-embeds run-generator \
		CONFIG=../../internal/pricing/regions.yaml \
		TEMPLATE=embed_template.go.tmpl \
		OUTPUT=../../internal/pricing

.PHONY: generate-goreleaser
generate-goreleaser: ## Generate .goreleaser.yaml from regions.yaml
	@echo "Generating GoReleaser config..."
	@$(MAKE) -C tools/generate-goreleaser run-generator \
		CONFIG=../../internal/pricing/regions.yaml \
		OUTPUT=../../.goreleaser.yaml

.PHONY: verify-regions
verify-regions: ## Verify region configuration and generated files
	@echo "Verifying region configuration..."
	@./scripts/verify-regions.sh

.PHONY: verify-embeds
verify-embeds: ## Verify embed template and fallback have matching variables
	@echo "Verifying embed template and fallback sync..."
	@TEMPLATE_VARS=$$(grep -oE 'var raw[A-Za-z0-9]+JSON' tools/generate-embeds/embed_template.go.tmpl | sort); \
	FALLBACK_VARS=$$(grep -oE 'var raw[A-Za-z0-9]+JSON' internal/pricing/embed_fallback.go | sort); \
	if [ "$$TEMPLATE_VARS" != "$$FALLBACK_VARS" ]; then \
		echo ""; \
		echo "❌ ERROR: Embed template and fallback have mismatched variables!"; \
		echo ""; \
		echo "Template (tools/generate-embeds/embed_template.go.tmpl):"; \
		echo "$$TEMPLATE_VARS" | sed 's/^/  /'; \
		echo ""; \
		echo "Fallback (internal/pricing/embed_fallback.go):"; \
		echo "$$FALLBACK_VARS" | sed 's/^/  /'; \
		echo ""; \
		echo "Both files must declare the same raw*JSON variables."; \
		echo "See CLAUDE.md 'Adding New AWS Services' for instructions."; \
		exit 1; \
	fi
	@echo "✓ Embed template and fallback are in sync"

.PHONY: lint
lint: verify-embeds ## Run golangci-lint (includes embed verification)
	@echo "Running linter..."
	@golangci-lint run --allow-parallel-runners ./...

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: test-integration
test-integration: ## Run integration tests (verify real pricing is embedded)
	@echo "Running integration tests..."
	@echo "Building binaries and testing gRPC server responses..."
	@go test -v -tags=integration ./internal/plugin/... -run TestIntegration_VerifyPricingEmbedded

.PHONY: build
build: ## ⚠️  Build with FALLBACK pricing (development only - do NOT release)
	@echo ""
	@echo "⚠️  WARNING: Building with fallback/dummy pricing (development mode)"
	@echo "   Only test SKUs (t3.micro, t3.small, gp2, gp3) have prices"
	@echo "   Real instance types will return \$$0 cost"
	@echo ""
	@echo "For real pricing data, use one of:"
	@echo "  make build-default-region      # Build us-east-1 with real pricing"
	@echo "  make build-region REGION=us-east-1  # Build any region with real pricing"
	@echo "  make build-all-regions         # Build all 12 regions"
	@echo ""
	@go build -ldflags "$(LDFLAGS)" -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

.PHONY: build-default-region
build-default-region: ## Build us-east-1 with real AWS pricing (RECOMMENDED)
	@echo "Building us-east-1 with real pricing..."
	@$(MAKE) build-region REGION=us-east-1

.PHONY: build-region
build-region: ## Build region-specific binary (usage: make build-region REGION=us-east-1)
	@echo "Building plugin for region $(REGION)..."
	@go build -ldflags "$(LDFLAGS)" -tags region_$(shell ./scripts/region-tag.sh $(REGION)) -o pulumicost-plugin-aws-public-$(REGION) ./cmd/pulumicost-plugin-aws-public

.PHONY: build-all-regions
build-all-regions: ## Build binaries for all supported regions
	@echo "Building all $(REGION_COUNT) region binaries..."
	@for region in $(REGIONS); do \
		echo "Building $$region..."; \
		go build -ldflags "$(LDFLAGS)" -tags region_$$(./scripts/region-tag.sh $$region) -o pulumicost-plugin-aws-public-$$region ./cmd/pulumicost-plugin-aws-public || exit 1; \
	done
	@echo "All region binaries built successfully!"
	@ls -lh pulumicost-plugin-aws-public-*

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f pulumicost-plugin-aws-public*
	@rm -rf dist/
