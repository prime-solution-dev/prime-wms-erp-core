SHELL := /usr/bin/fish

.PHONY: test test-integration tidy

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./internal/repositories/priceList

tidy:
	go mod tidy

