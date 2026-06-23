.PHONY: test run-help run-version

test:
	go test ./...

run-help:
	go run ./cmd/flowman --help

run-version:
	go run ./cmd/flowman version
