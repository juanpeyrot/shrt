include ./.env
export

.PHONY: run test build

run:
	go run ./cmd/api

test:
	go test ./internal/postgres_test/... -v

build:
	go build ./...