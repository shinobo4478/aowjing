# backend

ACMP API server + worker — Go, [chi](https://github.com/go-chi/chi) router,
[pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) for Postgres,
[asynq](https://github.com/hibiken/asynq) + Redis for the job queue.

## Layout

```
cmd/api/               HTTP server entrypoint
cmd/worker/            generation-queue worker entrypoint
internal/config/       env -> Config, .env loader
internal/database/     pgx connection pool
internal/auth/         single-admin login, DB-backed session cookie, middleware
internal/profiles/     Profile CRUD (+ the `provider` field)
internal/channels/     Channel CRUD (nested under a profile)
internal/prompttemplates/  Prompt template CRUD
internal/generate/     Generator interface + TextGenerator
internal/generations/  enqueue (handler) + execute (Runner) a generation
internal/queue/         asynq task names, payloads, enqueue client
internal/settings/     global key/value credential store
internal/server/       chi router, CORS, route mounting
db/schema.sql          DDL (source of truth for docker + sqlc)
db/query/              sqlc query definitions
```

## Auth

One admin user, credentials from `ADMIN_USERNAME` / `ADMIN_PASSWORD`. Login
mints a random opaque token; only its SHA-256 hash is stored in the `sessions`
table, and the raw token goes to the browser in an `HttpOnly` cookie
(`acmp_session`). Logout and expiry delete the row. Everything except
`/healthz` and `/auth/*` requires a live session.

Browsers must send the cookie: `fetch(url, { credentials: "include" })`.

## Generations

`POST /generations {promptTemplateId}` creates a `pending` row and enqueues a
job; **the worker must be running** to process it. The client polls
`GET /generations?profileId=…` until the status leaves `pending`.

## Local development

Prerequisites: Go 1.23+, Docker, [`sqlc`](https://docs.sqlc.dev/en/latest/overview/install.html).

```sh
cp .env.example .env      # defaults match docker-compose
```

The full stack is four processes. From the **repo root**, `./dev.ps1` (Windows
PowerShell) starts all of them in separate windows; `./dev.ps1 -Down` stops
Postgres + Redis. To run them by hand:

| Where | With `make` | Without `make` |
| --- | --- | --- |
| `backend/` | `make db-up` | `docker compose up -d` |
| `backend/` | `make run` | `go run ./cmd/api` |
| `backend/` | `make worker` | `go run ./cmd/worker` |
| `frontend/` | — | `npm run dev` |

If `go` isn't on PATH (terminal opened before installing it), use the full
path, e.g. `& 'C:\Program Files\Go\bin\go.exe' run ./cmd/api`.

Check it:

```sh
curl localhost:8080/healthz
# {"status":"ok","db":"ok"}
```

## Common tasks

| `make` target | Without `make` | Does |
| --- | --- | --- |
| `make run` | `go run ./cmd/api` | run the API on :8080 |
| `make worker` | `go run ./cmd/worker` | run the queue worker |
| `make build` | `go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker` | compile both binaries |
| `make test` | `go test ./...` | run tests |
| `make sqlc` | `sqlc generate` | regenerate DB code after editing `db/schema.sql` or `db/query/` |
| `make db-up` | `docker compose up -d` | start Postgres + Redis |
| `make db-down` | `docker compose down` | stop them (keep data) |
| `make db-reset` | `docker compose down -v && docker compose up -d` | wipe the DB volume and re-apply the schema |
