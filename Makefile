# All supported AWS regions (sourced from internal/pricing/regions.yaml)
# Uses Go-based YAML parser for robust parsing (replaces fragile awk/sed)
REGIONS_FILE := internal/pricing/regions.yaml
PARSE_REGIONS := cd tools/parse-regions && go mod download -x 2>/dev/null && go run . -config ../../$(REGIONS_FILE) -format lines
REGIONS := $(shell $(PARSE_REGIONS) -field name)
REGIONS_CSV := $(shell $(PARSE_REGIONS) -field name | tr '\n' ',' | sed 's/,$$//')
REGION_COUNT := $(words $(REGIONS))

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
generate-pricing: ## Generate pricing data for all regions
	@echo "Generating pricing data for all regions..."
	@go run ./tools/generate-pricing --regions $(REGIONS_CSV) --out-dir ./internal/pricing/data

.PHONY: generate-carbon-data
generate-carbon-data: ## Fetch CCF instance specs for carbon estimation
	@echo "Fetching Cloud Carbon Footprint instance specs..."
	@go run ./tools/generate-carbon-data --out-dir ./internal/carbon/data

.PHONY: generate-embeds
generate-embeds: ## Generate embed files from regions.yaml
	@echo "Generating embed files..."
	@cd tools/generate-embeds && go run . --config ../../internal/pricing/regions.yaml --template embed_template.go.tmpl --output ../../internal/pricing

.PHONY: generate-goreleaser
generate-goreleaser: ## Generate .goreleaser.yaml from regions.yaml
	@echo "Generating GoReleaser config..."
	@cd tools/generate-goreleaser && go run . --config ../../internal/pricing/regions.yaml --output ../../.goreleaser.yaml

.PHONY: verify-regions
verify-regions: ## Verify region configuration and generated files
	@echo "Verifying region configuration..."
	@./scripts/verify-regions.sh

.PHONY: lint
lint: ## Run golangci-lint
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
	@go build -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

.PHONY: build-default-region
build-default-region: ## Build us-east-1 with real AWS pricing (RECOMMENDED)
	@echo "Building us-east-1 with real pricing..."
	@$(MAKE) build-region REGION=us-east-1

.PHONY: build-region
build-region: ## Build region-specific binary (usage: make build-region REGION=us-east-1)
	@echo "Building plugin for region $(REGION)..."
	@go build -tags region_$(shell ./scripts/region-tag.sh $(REGION)) -o pulumicost-plugin-aws-public-$(REGION) ./cmd/pulumicost-plugin-aws-public

.PHONY: build-all-regions
build-all-regions: ## Build binaries for all supported regions
	@echo "Building all $(REGION_COUNT) region binaries..."
	@for region in $(REGIONS); do \
		echo "Building $$region..."; \
		go build -tags region_$$(./scripts/region-tag.sh $$region) -o pulumicost-plugin-aws-public-$$region ./cmd/pulumicost-plugin-aws-public || exit 1; \
	done
	@echo "All region binaries built successfully!"
	@ls -lh pulumicost-plugin-aws-public-*

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f pulumicost-plugin-aws-public*
	@rm -rf dist/
