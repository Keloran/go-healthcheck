-include .env
export

.PHONY: setup
setup: ## Get linting stuffs
	go get github.com/golangci/golangci-lint/cmd/golangci-lint
	go get golang.org/x/tools/cmd/goimports

.PHONY: test
test: lint ## Test the app
	go test \
		-v \
		-race \
		-bench=./... \
		-benchmem \
		-timeout=120s \
		-cover \
		-coverprofile=./test_coverage.txt \
		-bench=./... ./...

.PHONY: mocks
mocks: ## Generate the mocks
	go generate ./...

.PHONY: full
full: fmt lint test ## Clean, build, make sure its formatted, linted, and test it

.PHONY: lint
lint: ## Lint
	golangci-lint run --config golangci.yml

.PHONY: fmt
fmt: ## Formatting
	gofmt -w -s .
	goimports -w .
	go clean ./...

.PHONY: pre-commit
pre-commit: fmt lint ## Do formatting and linting

.PHONY: clean
clean: ## Clean
	go clean ./...
	rm -rf bin/${SERVICE_NAME}
