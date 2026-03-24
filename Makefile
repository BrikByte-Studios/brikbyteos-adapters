.PHONY: test deps

deps:
	go mod tidy
test:
	go test ./...