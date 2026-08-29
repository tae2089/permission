GO ?= go
GOFMT ?= gofmt
GOVULNCHECK ?= $(GO) tool govulncheck
GO_FILES ?= $(shell find . -type f -name '*.go' \
	-not -path './.git/*' \
	-not -path './_workspace/*' \
	-not -path './vendor/*' \
	| sort)

.PHONY: verify fmt-check architecture test test-race vet vuln

verify: fmt-check architecture test test-race vet vuln

fmt-check:
	@if [ -z "$(strip $(GO_FILES))" ]; then \
		echo "no Go files found"; \
		exit 1; \
	fi
	@unformatted="$$($(GOFMT) -l $(GO_FILES))"; \
	if [ -n "$$unformatted" ]; then \
		printf 'unformatted Go files:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

architecture:
	$(GO) run ./scripts/architecture

test:
	$(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...

vet:
	$(GO) vet ./...

vuln:
	$(GOVULNCHECK) ./...
