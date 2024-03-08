default: help

.PHONY: lint
lint: vet fmt ## Format and vet code

.PHONY: ci-lint
ci-lint: vet ## Check code format and vet code
## Specifically for CI, this will fail on any unformatted code.
	@test -n "$$(gofmt -l .)" \
		&& echo 'Files below need formatting, run `make fmt`:' \
		&& gofmt -l . \
		&& exit 1 \
		|| exit 0

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt: ## Format the code
	go fmt ./...

.PHONY: test
test: ## Run tests
	go test -cover ./...

.PHONY: help
help: Makefile ## Print this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; \
		{printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
