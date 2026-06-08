include ./.env
export

MIGRATIONS_PATH=./internal/db/migrations
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

.PHONY: run test build migrate-up migrate-down migrate-create

run:
	go run ./cmd/api

test:
	go test ./internal/db_test/... -v

build:
	go build ./...

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)