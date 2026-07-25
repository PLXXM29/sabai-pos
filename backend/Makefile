# Convenience targets. If you don't have `make`, the raw commands are in README.
.PHONY: run build ui ui-reset test test-all vet fmt sqlc tidy migrate-down

run:            ## run the API locally (needs a reachable DATABASE_URL)
	go run ./cmd/api

ui:             ## build the web app into the directory the binary embeds
	cd ../frontend && npm run build
	find internal/web/dist -mindepth 1 -not -name .gitignore -delete
	cp -r ../frontend/dist/. internal/web/dist/

ui-reset:       ## restore the committed placeholder page after `make ui`
	git checkout -- internal/web/dist

build: ui       ## binary that serves the UI as well as the API
	go build -o bin/api ./cmd/api

test:           ## unit tests (no Docker needed)
	go test ./...

test-all: test  ## + integration tests against a real Postgres (testcontainers)
	go test -tags=integration ./internal/store/...

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

sqlc:           ## regenerate type-safe DB code from queries/ + migrations/
	sqlc generate

tidy:
	go mod tidy

# The API applies pending migrations itself on boot, so there is no migrate-up
# target. Rolling one back is the case that still needs a tool.
DB_URL ?= postgres://sabai:change_me_dev_password@localhost:5433/sabai?sslmode=disable
migrate-down:   ## roll back the most recent migration
	docker run --rm -v "$(CURDIR)/migrations:/migrations" --network host migrate/migrate:v4.18.1 \
		-path /migrations -database "$(DB_URL)" down 1
