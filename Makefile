.PHONY: help lint test build clean

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

build: ## Build all binaries (no build tags - uses fallback)
	@echo "Building plugin..."
	@go build -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

build-region: ## Build region-specific binary (usage: make build-region REGION=us-east-1)
	@echo "Building plugin for region $(REGION)..."
	@go build -tags region_$(shell echo $(REGION) | tr '-' '_' | sed 's/us_east_1/use1/;s/us_west_2/usw2/;s/eu_west_1/euw1/') -o pulumicost-plugin-aws-public-$(REGION) ./cmd/pulumicost-plugin-aws-public

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f pulumicost-plugin-aws-public*
	@rm -rf dist/
