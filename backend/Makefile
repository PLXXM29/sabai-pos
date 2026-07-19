# Convenience targets. If you don't have `make`, the raw commands are in README.
.PHONY: run build test vet sqlc tidy migrate-up migrate-down

run:            ## run the API locally (needs a reachable DATABASE_URL)
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

sqlc:           ## regenerate type-safe DB code from queries/ + migrations/
	sqlc generate

tidy:
	go mod tidy

# DB migrations via the dockerised migrate CLI (no local install needed).
DB_URL ?= postgres://minimart:change_me_dev_password@localhost:5432/minimart?sslmode=disable
migrate-up:
	docker run --rm -v "$(CURDIR)/migrations:/migrations" --network host migrate/migrate:v4.18.1 \
		-path /migrations -database "$(DB_URL)" up

migrate-down:
	docker run --rm -v "$(CURDIR)/migrations:/migrations" --network host migrate/migrate:v4.18.1 \
		-path /migrations -database "$(DB_URL)" down 1
