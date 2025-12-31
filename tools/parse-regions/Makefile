# tools/parse-regions/Makefile

CONFIG ?= ../../internal/pricing/regions.yaml
FORMAT ?= lines
FIELD ?= name

.PHONY: run-parser
run-generator:
	go run . -config $(CONFIG) -format $(FORMAT) -field $(FIELD)
