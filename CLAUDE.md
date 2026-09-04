# Project: AI Content Management Platform

Personal-use content management system for creating and distributing AI-generated
video content across multiple channels/profiles and social platforms. Deployed to cloud.

## Stack

- **Frontend**: Next.js (App Router), TypeScript
- **UI library**: Ant Design (antd) — use its components for all admin
  screens (Table, Form, DatePicker, etc.) instead of building custom ones.
  Requires wrapping the app with `AntdRegistry` (from `@ant-design/nextjs-registry`)
  for the CSS-in-JS to work correctly with React Server Components — set this
  up as part of the initial Next.js scaffold, not later
- **Font**: Sukhumvit — keep using it across all new screens
- **Backend**: Go — router: chi or gin (keep it lightweight), DB access: sqlc
  (write raw SQL, generate type-safe Go code — prefer this over a heavy ORM
  since I'm still learning Go)
- **Database**: PostgreSQL
- **Queue/async jobs**: Redis + asynq
- **Object storage**: S3-compatible (Cloudflare R2 or similar) — thumbnails only,
  never store full video files
- **Auth**: DB-backed opaque session token, SHA-256-hashed in a `sessions`
  table, delivered as an `HttpOnly` cookie. Single admin from env. (Decided
  in Phase 1.)

## Architecture

- `frontend/` — Next.js app
- `backend/` — Go: `cmd/api` (HTTP server) + `cmd/worker` (asynq queue
  consumer), sharing `internal/`
- Backend and frontend are separate deployables, talk over REST/JSON +
  the session cookie
- Backend owns all business logic and DB access — frontend never talks to
  Postgres directly
- `db/schema.sql` is the single source of truth for DDL (feeds both the
  docker Postgres init and sqlc). No migration tool yet — schema changes to
  an existing DB need a hand-run `ALTER`.
- One-command local dev: `./dev.ps1` (repo root) starts Postgres + Redis,
  api, worker, and the frontend.

### Generator system

- Single `Generator` interface, multiple swappable implementations behind it
  — never write code elsewhere in the app against a specific provider
  directly
- `TextGenerator` — produces prompt text only, no external API cost. The
  base/fallback implementation.
- `FalVideoGenerator` — calls fal.ai (default model: Kling 3.0), produces an
  actual video. Same interface as `TextGenerator`. `FakeFalGenerator`
  (`AI_FAKE_FAL=1`) stands in for local dev without a key.
- The worker picks the generator per generation from the profile's
  `provider` field.
- Provider credentials (API keys/tokens) live in ONE central Settings
  table/screen, global — never duplicated per Profile. The fal.ai key is
  read fresh from that store on each run.
- Each Profile has a `provider` field selecting which Generator/model it
  defaults to — this is what future automation will read to decide how to
  generate for that profile

## Phase 1 (completed)

1. ✅ Auth — DB session cookie, single admin from env
2. ✅ Profile CRUD
3. ✅ Channel CRUD (nested under Profile, cascade delete)
4. ✅ Prompt template CRUD (tied to Profile)
5. ✅ Generate — runs a template through a provider + saves history
   (placeholder `MockGenerator`, replaced in Phase 2 item 1b)
6. ✅ Next.js UI for all of the above (antd, Sukhumvit font)

## Phase 2 scope (current focus — do not go beyond this list)

1. Generator system — ✅ **done**
   a. ✅ `Generator` interface
   b. ✅ `TextGenerator` (replaced `MockGenerator`)
   c. ✅ `FalVideoGenerator` (fal.ai, Kling 3.0) — real path needs a fal.ai
      key; verified end-to-end with `FakeFalGenerator`
   d. ✅ Central Settings screen for provider credentials
   e. ✅ `provider` field on Profile
2. ✅ **done** — Background job queue (Redis + asynq). `POST /generations`
   enqueues; `cmd/worker` runs it; the UI polls for the result.
3. ⬜ Connect ONE platform for posting — start with YouTube (most open API of
   the three), OAuth account connection + manual publish trigger.
   *Blocked on: a Google Cloud project + OAuth client credentials.*
4. ⬜ Basic statistics pulling from that one platform's analytics API,
   thumbnail only, no full video storage.

## Explicitly OUT of scope for Phase 2 (do not build yet)

- TikTok / Instagram integration — after YouTube is solid
- AI-automated prompt suggestions (auto-picking which provider/prompt to run
  — item 1e only stores the *setting*, it does not trigger anything
  automatically yet)
- Multi-language pipeline

## Conventions

- Go: standard project layout (`cmd/`, `internal/`), table-driven tests,
  `gofmt`/`golangci-lint` clean
- Commit small, working increments — don't batch multiple features into one
  commit
- Explain Go idioms briefly when introducing something non-obvious — I'm
  learning Go as we go
- Ask before adding a new external dependency
- UI: stick to Ant Design components — don't introduce Tailwind, shadcn/ui,
  or hand-rolled CSS for screens that antd already covers
- PRs target `main` directly (no stacked PRs — they mis-merge when the parent
  merges first)
