\# Project: AI Content Management Platform



Personal-use content management system for creating and distributing AI-generated

video content across multiple channels/profiles and social platforms. Deployed to cloud.



\## Stack



\- \*\*Frontend\*\*: Next.js (App Router), TypeScript

\- \*\*Backend\*\*: Go — router: chi or gin (keep it lightweight), DB access: sqlc

&#x20; (write raw SQL, generate type-safe Go code — prefer this over a heavy ORM

&#x20; since I'm still learning Go)

\- \*\*Database\*\*: PostgreSQL

\- \*\*Queue/async jobs\*\*: Redis + asynq (for Phase 2+, not needed yet in Phase 1)

\- \*\*Object storage\*\*: S3-compatible (Cloudflare R2 or similar) — thumbnails only,

&#x20; never store full video files

\- \*\*Auth\*\*: session or JWT based, backend-issued (decide during Phase 1 auth task)



\## Architecture



\- `frontend/` — Next.js app

\- `backend/` — Go API server (and later, worker binary)

\- Backend and frontend are separate deployables, talk over REST/JSON

\- Backend owns all business logic and DB access — frontend never talks to

&#x20; Postgres directly



\## Phase 1 scope (current focus — do not go beyond this list)



1\. Auth (login/session for a single admin user is enough for now — no

&#x20;  multi-tenant complexity yet)

2\. Profile CRUD — a Profile represents one "channel persona" / content niche

3\. Channel CRUD — nested under a Profile

4\. Prompt template CRUD — store reusable prompt templates, associate with a

&#x20;  Profile

5\. Single AI provider integration — pick ONE video/prompt AI provider to

&#x20;  start, manual "generate" trigger only (no automation, no queue yet)

6\. Basic Next.js UI for all of the above (forms + lists, no polish needed)



\## Explicitly OUT of scope for Phase 1 (do not build yet)



\- Platform posting/OAuth integration (YouTube, TikTok, Instagram)

\- Multi-AI-provider abstraction — hardcode one provider for now

\- Statistics/analytics dashboards

\- Background job queue / async workers

\- AI-automated prompt suggestions

\- Multi-language pipeline



\## Conventions



\- Go: standard project layout (`cmd/`, `internal/`), table-driven tests,

&#x20; `gofmt`/`golangci-lint` clean

\- Commit small, working increments — don't batch multiple features into one

&#x20; commit

\- Explain Go idioms briefly when introducing something non-obvious — I'm

&#x20; learning Go as we go

\- Ask before adding a new external dependency

