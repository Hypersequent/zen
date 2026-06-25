GO ?= go

# Go modules in the workspace (see go.work). golangci-lint and `go test ./...`
# don't cross module boundaries, so each module is handled explicitly.
MODULES = . ./custom/decimal ./custom/optional

linters-install:
	@golangci-lint --version >/dev/null 2>&1 || { \
		echo "installing linting tools..."; \
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.12.2; \
	}

lint: linters-install
	@for dir in $(MODULES); do \
		echo "==> golangci-lint run ($$dir)"; \
		(cd $$dir && golangci-lint run) || exit 1; \
	done

format: linters-install
	@for dir in $(MODULES); do \
		echo "==> golangci-lint fmt ($$dir)"; \
		(cd $$dir && golangci-lint fmt) || exit 1; \
	done

test:
	@for dir in $(MODULES); do \
		echo "==> go test ($$dir)"; \
		$(GO) test -C $$dir -cover -race ./... || exit 1; \
	done

# Regenerates golden files. Only the root module uses the golden fixtures.
test-update:
	GOLDEN_UPDATE=true $(GO) test ./...

docker-test:
	./docker-test.sh

.PHONY: lint format test test-update linters-install docker-test
