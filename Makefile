.PHONY: build run test test-verbose test-cover test-package lint clean

build:
	go build -o goperf ./cmd/goperf

run:
	go run ./cmd/goperf

# Run all tests with race condition detection
test:
	go test -race ./...

# Run all tests with detailed output per test
test-verbose:
	go test -race -v ./...

# Run all tests with code coverage report
test-cover:
	go test -race -cover ./...

# Run tests for a single package, e.g.: make test-package PKG=./internal/worker/
test-package:
	go test -race -v $(PKG)

lint:
	gofmt -w .
	go vet ./...

clean:
	rm -f goperf
