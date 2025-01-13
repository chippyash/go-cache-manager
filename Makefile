.PHONY: help
help:  ## Print the help documentation
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Run unit tests
	go test ./adapter/valkey ./adapter/memory

.PHONY: license-check
license-check: ## Run the Go license checker
	go install github.com/google/go-licenses@latest
	go-licenses report --ignore github.com/chippyash/go-cache-manager ./... > licenses.csv