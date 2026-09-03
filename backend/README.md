# backend

ACMP API server — Go, [chi](https://github.com/go-chi/chi) router,
[pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) for Postgres.

## Layout

```
cmd/api/            server entrypoint (config, DB, graceful shutdown)
internal/config/    env -> Config
internal/database/  pgx connection pool
internal/server/    chi router + handlers
db/schema.sql       DDL (source of truth for docker + sqlc)
db/query/           sqlc query definitions
```

## Local development

Prerequisites: Go 1.23+, Docker, and [`sqlc`](https://docs.sqlc.dev/en/latest/overview/install.html).

```sh
cp .env.example .env      # defaults match docker-compose
make db-up                # start Postgres (applies db/schema.sql on first run)
make run                  # start the API on :8080
```

Check it:

```sh
curl localhost:8080/healthz
# {"status":"ok","db":"ok"}
```

## Common tasks

| Command | Does |
| --- | --- |
| `make run` | run the server |
| `make build` | compile to `bin/api` |
| `make test` | run tests |
| `make sqlc` | regenerate DB code after editing `db/schema.sql` or `db/query/` |
| `make db-reset` | wipe the DB volume and re-apply the schema |
