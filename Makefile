SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := verify
GO ?= go
IQKVS_MAXPROCS ?= 16
ifneq ($(IQKVS_MAXPROCS),0)
export GOMAXPROCS := $(IQKVS_MAXPROCS)
endif

.PHONY: verify fmt-check vet test demo demo-svg demo-check
verify: fmt-check vet test demo-check
fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
vet:
	$(GO) vet -p 1 ./...
test:
	$(GO) test -race -p 1 ./...
demo:
	$(GO) run ./examples/demo
demo-svg:
	$(GO) run ./examples/demo -svg examples/demo/hero.svg
demo-check:
	@out=$$(mktemp); trap 'rm -f "$$out"' EXIT; $(GO) run ./examples/demo -svg "$$out"; cmp "$$out" examples/demo/hero.svg
