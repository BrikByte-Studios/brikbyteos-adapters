.PHONY: test deps fmt

deps:
	go mod tidy
test:
	go test ./...

# Format all Go source files in the repository.
fmt:
	go fmt ./...
