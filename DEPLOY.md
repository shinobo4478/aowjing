# Deploy (personal, single host)

One small VPS runs the whole stack in Docker. Caddy is the only thing on the
public internet; it terminates HTTPS and routes:

```
                 ┌────────── your VPS ──────────────────────┐
  phone/laptop   │                                          │
  ───HTTPS──▶  caddy ──/api/*──▶ api      ──▶ db (postgres)  │
                 │        └────▶ web      ──▶ redis          │
                 │                  worker ──▶ db, redis, fal.ai
                 └──────────────────────────────────────────┘
```

Frontend and API share one origin (`https://your-domain` and
`https://your-domain/api`), so the session cookie just works and there's no
CORS dance.

---

## 1. What you need

- A VPS: **1 vCPU / 2 GB RAM / 20 GB disk** is enough (Hetzner CX22, DO basic
  droplet, Vultr, etc.). Ubuntu 22.04/24.04 LTS.
- A domain name, with a DNS **A record** (and AAAA if you have IPv6) pointing
  at the VPS IP. A subdomain like `acmp.yourdomain.com` is fine.
- Ports **80** and **443** open to the world (needed for the Let's Encrypt
  challenge and for you).

## 2. One-time server setup

SSH in as a sudo user, then install Docker Engine + the compose plugin:

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker   # or log out and back in
```

Verify: `docker compose version`.

## 3. Get the code and configure

```bash
git clone https://github.com/shinobo4478/aowjing.git
cd aowjing
cp .env.prod.example .env.prod
nano .env.prod
```

Fill in `.env.prod`:

| var | set to |
| --- | --- |
| `SITE_ADDRESS` | `acmp.yourdomain.com` (the DNS name you pointed here) |
| `POSTGRES_PASSWORD` | a long random string — `openssl rand -base64 24` |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | your login; password is required |
| `CORS_ORIGIN` | `https://acmp.yourdomain.com` (scheme + host, no slash) |
| `AI_FAKE_FAL` | `1` for now (no fal.ai key yet); `0` once you add one |

Everything else can stay as-is.

## 4. Bring it up

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

First run builds the two images and pulls Postgres/Redis/Caddy (~2–4 min).
Caddy fetches a TLS cert automatically once DNS resolves to this box.

Check it:

```bash
docker compose -f docker-compose.prod.yml ps
curl -sS https://acmp.yourdomain.com/api/healthz    # -> {"status":"ok","db":"ok"}
```

Open `https://acmp.yourdomain.com` on your phone and log in with the admin
credentials from `.env.prod`.

> **Tip:** put the long compose command in a shell alias or a one-line script:
> `alias dc='docker compose -f docker-compose.prod.yml --env-file .env.prod'`
> then `dc up -d --build`, `dc logs -f`, `dc ps`, `dc down`.

## 5. Updating to a new version

```bash
cd aowjing
git pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

Compose recreates only the containers whose image changed. Postgres data
survives in the `acmp_pgdata` volume.

## 6. Schema changes

`backend/db/schema.sql` runs **only once**, when the Postgres volume is first
created. There's no migration tool yet, so after changing the schema you
hand-apply the diff to the live DB:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec db psql -U acmp -d acmp -c "ALTER TABLE ... ;"
```

Or, if you don't care about the data yet, wipe and re-init:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod down
docker volume rm acmp_pgdata
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

## 7. Backups

Thumbnails live in S3/R2 (not here); the only stateful thing on the box is
Postgres. Dump it on a schedule:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec -T db pg_dump -U acmp -d acmp | gzip > acmp-$(date +%F).sql.gz
```

Drop that into a cron job and copy the file off-box (rclone to R2, scp, …).

Restore:

```bash
gunzip -c acmp-2026-01-01.sql.gz | docker compose -f docker-compose.prod.yml \
  --env-file .env.prod exec -T db psql -U acmp -d acmp
```

## 8. Logs & ops

```bash
dc logs -f api          # or worker / web / caddy / db
dc restart api
dc down                 # stop everything (data kept)
dc down -v              # stop + delete volumes (Postgres + Caddy certs) — careful
```

## 9. Adding the fal.ai key later

No redeploy needed. Set `AI_FAKE_FAL=0` in `.env.prod`, then
`dc up -d` (recreates api + worker), then paste the key into the **Settings**
screen in the app. The worker reads it from the DB on each run.

## 10. Troubleshooting

| symptom | check |
| --- | --- |
| Cert never issues / 526 | DNS really points here? `dig acmp.yourdomain.com`. Port 80 open? `dc logs caddy` |
| `502` from `/api` | `dc logs api` — usually `ADMIN_PASSWORD` unset or DB not healthy yet |
| Login works then 401 on every call | `COOKIE_SECURE=true` but you're on plain HTTP, or `CORS_ORIGIN` mismatch |
| worker does nothing | `dc logs worker`; is `redis` healthy? is a generation actually enqueued? |
| build OOM on a 1 GB box | add 1 GB swap, or `docker build` the images on your laptop and push to a registry |

## Alternatives (not scaffolded here)

- **Railway / Render / Fly.io** — managed Postgres + Redis add-ons, deploy each
  service from the repo, skip Caddy and the VPS entirely. More hands-off,
  ~$10–20/mo, and secrets live in their dashboard. Worth it if you'd rather not
  babysit a server.
- **Split**: frontend on Vercel, backend + DB on a VPS or Fly. Then you're back
  to cross-origin cookies — set `COOKIE_SECURE=true` and a precise
  `CORS_ORIGIN`, and point `NEXT_PUBLIC_API_BASE_URL` at the API domain.
