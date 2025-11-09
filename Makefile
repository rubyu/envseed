SHELL := /bin/bash
GO ?= go
DIST_DIR ?= dist
BIN_NAME ?= envseed
CMD_PKG ?= ./cmd/envseed

.PHONY: all build docs test test-integration test-sandbox test-fuzz check check-evt clean pre-commit

docs:
	$(GO) generate ./internal/envseed

build:
	@mkdir -p $(DIST_DIR)
	$(GO) build -o $(DIST_DIR)/$(BIN_NAME) $(CMD_PKG)

test:
	$(GO) test ./...

test-integration:
	$(GO) test -tags=integration ./...

test-sandbox:
	$(GO) test -tags=sandbox ./...

test-fuzz:
	@pkgs=$$($(GO) list ./...); \
	found=0; \
	for pkg in $$pkgs; do \
		fuzzes=$$($(GO) test -run=^$$ -list=^Fuzz $$pkg | grep '^Fuzz' || true); \
		if [ -n "$$fuzzes" ]; then \
			found=1; \
			for fuzz in $$fuzzes; do \
				echo "==> fuzzing $$pkg::$$fuzz"; \
				$(GO) test -run=^$$ -fuzz=$$fuzz -fuzztime=10m $$pkg || exit $$?; \
			done; \
		fi; \
	done; \
	if [ $$found -eq 0 ]; then \
		echo "no fuzz tests found"; \
	fi

check:
	@fmt_out=$$(gofmt -l $$($(GO) list -f '{{.Dir}}' ./...)); \
	if [ -n "$$fmt_out" ]; then \
		echo "gofmt needed for:"; \
		echo "$$fmt_out"; \
		exit 1; \
	fi
	$(GO) vet ./...
	$(MAKE) check-evt

check-evt:
	@scripts/check-evt.sh

pre-commit: docs check check-evt test test-sandbox test-integration

clean:
	rm -rf $(DIST_DIR)
