.PHONY: build run test lint clean

build:
	go build -o goperf ./cmd/goperf

run:
	go run ./cmd/goperf

test:
	go test -race ./...

lint:
	gofmt -w .
	go vet ./...

clean:
	rm -f goperf
