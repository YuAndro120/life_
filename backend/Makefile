# Локальная разработка бэкенда. Требуется: docker compose -f ../infra/docker-compose.dev.yml up -d
export DATABASE_URL ?= postgres://life:life_dev@127.0.0.1:5432/life?sslmode=disable
export TEST_DATABASE_URL ?= $(DATABASE_URL)
MIGRATIONS = internal/db/migrations

.PHONY: migrate migrate-down seed run generate test test-integration vet

migrate:
	goose -dir $(MIGRATIONS) postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS) postgres "$(DATABASE_URL)" down

seed:
	go run ./cmd/seed -yes

run:
	go run ./cmd/api

generate:
	sqlc generate

vet:
	gofmt -l . | (! grep .) && go vet ./...

test:
	go test ./...

test-integration: migrate seed
	go test -count=1 ./internal/store/
