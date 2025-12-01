# All supported AWS regions (sourced from internal/pricing/regions.yaml)
REGIONS_FILE := internal/pricing/regions.yaml
REGIONS := $(shell awk '/name:/ {gsub(/.*name: /, ""); print}' $(REGIONS_FILE))
REGIONS_CSV := $(shell awk '/name:/ {gsub(/.*name: /, ""); print}' $(REGIONS_FILE) | tr '\n' ',' | sed 's/,$$//')
REGION_COUNT := $(words $(REGIONS))

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
develop: ensure generate-pricing ## Setup development environment (install deps + generate pricing data)
	@echo "Development environment ready"

.PHONY: generate-pricing
generate-pricing: ## Generate pricing data for all regions
	@echo "Generating pricing data for all regions..."
	@go run ./tools/generate-pricing --regions $(REGIONS_CSV) --out-dir ./internal/pricing/data

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
	@golangci-lint run ./...

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: build
build: ## Build plugin (no build tags - uses fallback)
	@echo "Building plugin..."
	@go build -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

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
