default: help

.PHONY: lint
lint: vet fmt ## Format and vet the code

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
