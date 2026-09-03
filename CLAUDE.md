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

## Phase 1 scope (current focus — do not go beyond this list)

1. Auth (login/session for a single admin user is enough for now — no
   multi-tenant complexity yet)
2. Profile CRUD — a Profile represents one "channel persona" / content niche
3. Channel CRUD — nested under a Profile
4. Prompt template CRUD — store reusable prompt templates, associate with a
   Profile
5. Single AI provider integration — pick ONE video/prompt AI provider to
   start, manual "generate" trigger only (no automation, no queue yet)
6. Basic Next.js UI for all of the above (forms + lists, no polish needed)

## Explicitly OUT of scope for Phase 1 (do not build yet)

- Platform posting/OAuth integration (YouTube, TikTok, Instagram)
- Multi-AI-provider abstraction — hardcode one provider for now
- Statistics/analytics dashboards
- Background job queue / async workers
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