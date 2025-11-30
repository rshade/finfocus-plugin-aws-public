.PHONY: help lint test build clean ensure develop generate-pricing build-region build-all-regions

# All supported AWS regions
REGIONS := us-east-1 us-west-1 us-west-2 us-gov-west-1 us-gov-east-1 eu-west-1 ap-southeast-1 ap-southeast-2 ap-northeast-1 ap-south-1 ca-central-1 sa-east-1

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

ensure: ## Install development dependencies (goreleaser, golangci-lint)
	@echo "Installing development dependencies..."
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed successfully"

develop: ensure generate-pricing ## Setup development environment (install deps + generate pricing data)
	@echo "Development environment ready"

generate-pricing: ## Generate pricing data for all regions
	@echo "Generating pricing data for all regions..."
	@go run ./tools/generate-pricing --regions $(shell echo $(REGIONS) | tr ' ' ',') --out-dir ./internal/pricing/data

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

build: ## Build plugin (no build tags - uses fallback)
	@echo "Building plugin..."
	@go build -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

build-region: ## Build region-specific binary (usage: make build-region REGION=us-east-1)
	@echo "Building plugin for region $(REGION)..."
	@go build -tags region_$(shell ./scripts/region-tag.sh $(REGION)) -o pulumicost-plugin-aws-public-$(REGION) ./cmd/pulumicost-plugin-aws-public

build-all-regions: ## Build binaries for all supported regions
	@echo "Building all $(words $(REGIONS)) region binaries..."
	@for region in $(REGIONS); do \
		echo "Building $$region..."; \
		go build -tags region_$$(./scripts/region-tag.sh $$region) -o pulumicost-plugin-aws-public-$$region ./cmd/pulumicost-plugin-aws-public || exit 1; \
	done
	@echo "All region binaries built successfully!"
	@ls -lh pulumicost-plugin-aws-public-*

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f pulumicost-plugin-aws-public*
	@rm -rf dist/
