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
- **Queue/async jobs**: Redis + asynq (for Phase 2+, not needed yet in Phase 1)
- **Object storage**: S3-compatible (Cloudflare R2 or similar) — thumbnails only,
  never store full video files
- **Auth**: session or JWT based, backend-issued (decide during Phase 1 auth task)

## Architecture

- `frontend/` — Next.js app
- `backend/` — Go API server (and later, worker binary)
- Backend and frontend are separate deployables, talk over REST/JSON
- Backend owns all business logic and DB access — frontend never talks to
  Postgres directly

## Phase 1 (completed)

1. ✅ Auth — DB session cookie, single admin from env
2. ✅ Profile CRUD
3. ✅ Channel CRUD (nested under Profile, cascade delete)
4. ✅ Prompt template CRUD (tied to Profile)
5. ✅ Generate — runs a template through a provider + saves history, currently
   wired to a `MockGenerator` (not a real AI provider call yet)
6. ✅ Next.js UI for all of the above (antd, Sukhumvit font)

## Phase 2 scope (current focus — do not go beyond this list)

1. Swap `MockGenerator` for one real AI video/prompt provider (this was
   deferred from Phase 1 — do this first, before any new feature)
2. Background job queue (Redis + asynq) — move "generate" off the sync
   request path onto a worker
3. Connect ONE platform for posting — start with YouTube (most open API of
   the three), OAuth account connection + manual publish trigger
4. Basic statistics pulling from that one platform's analytics API, thumbnail
   only, no full video storage

## Explicitly OUT of scope for Phase 2 (do not build yet)

- TikTok / Instagram integration — after YouTube is solid
- Multi-AI-provider abstraction — still hardcoded to the one provider from
  item 1 above
- AI-automated prompt suggestions
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