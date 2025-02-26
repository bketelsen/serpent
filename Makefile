SHELL = /bin/bash
.ONESHELL:

.PHONY: lint
lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

.PHONY: test
test:
	go test -timeout=3m -race .
