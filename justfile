run:
  go run ./cmd/fin

build:
  go build -o ./bin/fin ./cmd/fin

test:
  go test -v -race -cover ./...

lint:
  golangci-lint run

ci: lint build test
