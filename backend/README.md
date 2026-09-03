# backend

ACMP API server — Go, [chi](https://github.com/go-chi/chi) router,
[pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) for Postgres.

## Layout

```
cmd/api/            server entrypoint (config, DB, graceful shutdown)
internal/config/    env -> Config
internal/database/  pgx connection pool
internal/auth/      single-admin login, DB-backed session cookie, middleware
internal/profiles/  Profile CRUD (session-protected)
internal/server/    chi router, CORS, route mounting
db/schema.sql       DDL (source of truth for docker + sqlc)
db/query/           sqlc query definitions
```

## Auth

One admin user, credentials from `ADMIN_USERNAME` / `ADMIN_PASSWORD`. Login
mints a random opaque token; only its SHA-256 hash is stored in the `sessions`
table, and the raw token goes to the browser in an `HttpOnly` cookie
(`acmp_session`). Logout and expiry delete the row, so access is revoked at
once. Everything except `/healthz` and `/auth/*` requires a live session.

| Method | Path | |
| --- | --- | --- |
| POST | `/auth/login` | `{username,password}` -> 200 + `Set-Cookie`, or 401 |
| POST | `/auth/logout` | 204, clears the cookie and the session row |
| GET | `/auth/me` | 200 `{"user":{…}}` when signed in, else 401 |

Browsers must send the cookie: `fetch(url, { credentials: "include" })`.

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
