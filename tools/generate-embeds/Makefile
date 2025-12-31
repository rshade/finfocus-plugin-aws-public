# tools/generate-embeds/Makefile

CONFIG ?= ../../internal/pricing/regions.yaml
TEMPLATE ?= embed_template.go.tmpl
OUTPUT ?= ../../internal/pricing

.PHONY: run-generator
run-generator:
	go run . --config $(CONFIG) --template $(TEMPLATE) --output $(OUTPUT)
