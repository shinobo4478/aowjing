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
3. ✅ **done** — Connect ONE platform for posting: YouTube OAuth connect
   (per Channel) + manual publish of a video generation. Uploads from an
   unverified Google project stay private until verified.
4. ⬜ Basic statistics pulling from that one platform's analytics API,
   thumbnail only, no full video storage.

## Phase 3 scope — Shopee products ("ปักตะกร้า") — Phase A done, B blocked

Attach up to **2 Shopee products per clip** so videos can carry affiliate
baskets. Decisions made with the owner:

- **Central catalog**, not per-profile: a product (link, price, commission)
  belongs to the product, not to a channel persona. One Products page; any
  profile's clip can pick from it.
- Products attach to a **generation** (the clip), max 2, enforced in the DB
  (position 1–2, unique per clip) and in the API.
- **Platform-agnostic:** the placement belongs to the clip, not a channel.
  Each platform renders it its own way — YouTube: a product block in the
  description + a checklist for tagging in YouTube Studio (no API for
  tagging has been found; verify); Facebook etc. later: links in the caption.
- No Shopee account yet, so phase A is manual: paste the product URL and
  your own affiliate link. Never scrape Shopee.
- An affiliate disclosure line goes on every description that has products.

Phase A — ✅ **done**, verified live against Postgres:
- catalog (table, Shopee link parser, CRUD API, Products page with search
  and "where is it pinned")
- pin up to 2 products to a clip (`generation_products`, cap enforced by the
  primary key), from the Generations row or when pressing Generate
- prompt templates can use `{{product.name}}` / `.price` / `.note` (and
  `{{product2.…}}`); the products are chosen at Generate time
- publish dialog: editable product block (Thai, with an affiliate disclosure
  when any link is an affiliate link) + a "Tag in YouTube Studio" checklist

Phase B — ⬜ blocked on a Shopee Affiliate account + Open API credentials
(and the Thai docs, which have not been read yet — don't code the client
from third-party descriptions): API client with credentials in Settings,
auto-fetch product info, affiliate short links with the upload id as subId,
conversion stats. Also not built because nothing consumes it yet: a Facebook
caption renderer (sits beside `internal/youtube/block.go` when Facebook is
connected).

## Phase 4 scope — Video studio (3 styles) — PROPOSED, awaiting owner approval

How a clip actually gets made. A **separate module** (`internal/studio`), not
tied to Shopee: each Profile picks a **video style** — `3d` (Pixar-like),
`2d`, `realistic` — and the style decides how its clips are produced.
Decisions made with the owner:

- **Start with `3d` on Kling, image-to-video.** 2D and realistic come after
  3D is proven. Style is stored on the Profile (`video_style`) and is
  separate from `provider` (`text`/`fal`).
- A style is a **recipe kept in one place** (registry), not scattered
  conditionals: model ids, the motion-prompt suffix (2D needs "flat shading,
  no 3D look"), and the audio route. Swapping a model should not touch
  handlers or the UI. Provider-specific code stays in `internal/generate`;
  credentials stay in Settings.
- **Workflow shape** (same for all styles): script/shot list → reference
  image → one approved keyframe per shot → image-to-video per shot (motion
  only in the prompt) → join + sound. The first version is **semi-manual**:
  the owner supplies the image URL (real product photo for Shopee clips, or a
  keyframe made elsewhere).
- **Thai voice-over:** none of the models' built-in audio is confirmed for
  Thai (Veo 3.1's documented languages exclude it). So clips are generated
  **silent** (`generate_audio: false` — the fal default is `true` and costs
  more) and Thai narration is added separately by TTS. Voice-over style only:
  no on-camera talking, there is no Thai lip-sync.
- Budget (prices are third-party figures, verify on fal.ai first): about
  $0.48 per 5 s shot and ≈ $2.9 per 30 s video on Kling v3 Standard without
  retakes; ≈ $4.4–5.8 with retakes. Show an estimate before generating.

Kling v3 image-to-video on fal (read from fal's model pages, 2026-11-26):
model `fal-ai/kling-video/v3/standard/image-to-video`; the image field is
`start_image_url` (not `image_url`); also `prompt`, `duration` ("3"–"15"),
`generate_audio`, `end_image_url`, `negative_prompt`, `cfg_scale`; output
`{"video":{"url":…}}`. Confirmed from fal's OpenAPI + queue docs (2026-11-26):
same queue pattern (`POST https://queue.fal.run/<model>`, `Authorization: Key …`,
use the `status_url`/`response_url` the submit reply returns — a hand-built URL
with the full model path gives 405); `start_image_url` is required; the image
may be a public URL or a base64 data URI (we only accept public https URLs);
limits 50 MB, min 300×300 px, aspect ratio 0.4–2.5; `duration` defaults "5" and
`generate_audio` defaults true. **Still unverified against a real key:** the
real request has not been run (only the request body is unit-tested).
**Price discrepancy:** fal's pricing page lists the Standard image-to-video
at $0.14/s (no audio on/off split stated), higher than the $0.084/s used for
the budget above — re-check on the fal dashboard before trusting the estimate
(a 5 s shot would be ≈ $0.70, not $0.42).

Order of work (each a small PR):
1. `image_url` on a generation + `FalVideoGenerator` uses the image-to-video
   model when it is present (`generate_audio:false`). **Backend done** (`FAL_IMAGE_MODEL`,
   `generate.Request`, `POST /generations` takes `imageUrl`, fal profiles only;
   existing DBs need `ALTER TABLE generations ADD COLUMN image_url text NOT NULL DEFAULT '';`).
   **UI done:** Generate on a fal profile asks for an optional starting-image link
   (`GenerateModal`); the clip row shows it. Still to do: the first real fal run
   (needs a key) and re-checking the price.
   **Upload from the owner's computer** (decided with the owner: keep in Postgres
   for now, move to R2 when it exists): `POST /images` (multipart `file`, JPG/PNG,
   ≤ 8 MB, ≥ 300×300 px, ratio 0.4–2.5), `GET /images/{id}`, and `POST /generations`
   takes `imageId` instead of `imageUrl`. The worker sends the picture to fal as a
   base64 data URI (fal's docs say data URIs are accepted; **unverified with a real
   key**, fal states no size cap for them). Deleting a clip deletes its picture if
   nothing else uses it; uploads never used by a clip, and pictures of clips removed
   with their profile, stay until cleaned by hand. Existing DBs need
   `CREATE TABLE images (…)` and `ALTER TABLE generations ADD COLUMN image_id uuid
   REFERENCES images (id) ON DELETE SET NULL;` (see `db/schema.sql`). The
   dialog has a "Link / From this computer" switch (`GenerateModal`); the file is
   uploaded when Generate is pressed, and the clip row shows its thumbnail.
2. `video_style` on Profile + the recipe registry (`3d` only).
3. Shots: a generation that is a set of ordered clips.
4. Keyframe generation (image model on fal) instead of pasting URLs.
5. Thai voice-over (TTS provider chosen after the owner listens to Thai
   samples from each candidate).
6. Join clips + audio. **Conflicts with a rule above:** joining makes a full
   video file, and this project never stores full videos. Until the owner
   decides (allow it, or keep joining manual in CapCut), this stays manual and
   needs ffmpeg (a new dependency — ask first).
7. `2d` and `realistic` recipes.

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
