.PHONY: run worker build test tidy sqlc db-up db-down db-reset

# Run the API server (reads .env).
run:
	go run ./cmd/api

# Run the generation-queue worker (reads .env). Needs Redis (make db-up).
worker:
	go run ./cmd/worker

# Compile both binaries to ./bin.
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

tidy:
	go mod tidy

# Regenerate type-safe DB code from db/schema.sql + db/query/*.sql.
sqlc:
	sqlc generate

# Local Postgres lifecycle.
db-up:
	docker compose up -d

db-down:
	docker compose down

# Wipe and recreate the DB (re-applies db/schema.sql).
db-reset:
	docker compose down -v
	docker compose up -d
