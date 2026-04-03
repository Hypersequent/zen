GOCMD=GO111MODULE=on go

linters-install:
	@golangci-lint --version >/dev/null 2>&1 || { \
		echo "installing linting tools..."; \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s v1.63.4; \
	}

lint: linters-install
	golangci-lint run

test:
	$(GOCMD) test -cover -race ./...

test-update:
	GOLDEN_UPDATE=true $(GOCMD) test ./...


docker-test:
	./docker-test.sh

bench:
	$(GOCMD) test -bench=. -benchmem ./...

.PHONY: test test-update lint linters-install docker-test bench
