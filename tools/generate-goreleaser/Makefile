# tools/generate-goreleaser/Makefile

CONFIG ?= ../../internal/pricing/regions.yaml
OUTPUT ?= ../../.goreleaser.yaml

.PHONY: run-generator
run-generator:
	go run . --config $(CONFIG) --output $(OUTPUT)
