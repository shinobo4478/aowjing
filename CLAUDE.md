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
`generate_audio` defaults true. **Verified with a real key (2026-11-26):** one
5 s image-to-video clip from an uploaded picture (sent as a base64 data URI)
succeeded on `fal-ai/kling-video/v3/standard/image-to-video` with
`generate_audio:false`; the owner judged the clip OK. It cost **$0.42**
(= $0.084/s), so the budget above holds — the $0.14/s on fal's pricing page is
presumably the audio-on rate. The clip took ~8 minutes from Generate to done
(the UI polls for 20 minutes). **Link lifetime (from fal's docs, 2026-11-26):**
generated files are kept "at least 7 days by default" (fal's FAQ; the retention
page gives no number), then deleted for good; a request header
`X-Fal-Object-Lifecycle-Preference: {"expiration_duration_seconds": N}` (null =
never) sets it per call — we do not send it yet. Download clips you want within
7 days. **Still unverified:** the 3D wording on a text-to-video clip (the image
clip adds no wording).

Order of work (each a small PR):
1. `image_url` on a generation + `FalVideoGenerator` uses the image-to-video
   model when it is present (`generate_audio:false`). **Backend done** (model now from the recipe, item 2;
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
2. `video_style` on Profile + the recipe registry (`3d` only). **Done.**
   `internal/studio` holds the recipes (fal model ids + prompt wording); the
   Profile stores only the style name. The worker takes its fal models from the
   recipe, so the `FAL_MODEL` / `FAL_IMAGE_MODEL` env vars are gone. The style's
   wording is added to the prompt when the clip is created (so the stored prompt
   is exactly what fal receives): a text-to-video clip gets `TextSuffix`; a clip
   that starts from a picture gets `ImageSuffix` (empty for `3d`, so a real
   product photo is never restyled). The 3D wording is a starting guess, not yet
   tried on real clips. Existing DBs need
   `ALTER TABLE profiles ADD COLUMN video_style text NOT NULL DEFAULT '3d';`.
   Adding a style = a constant + a recipe entry in `studio.go` + an option in
   `frontend/lib/types.ts`. The audio route joins the recipe with item 5.
3. Shots: a generation that is a set of ordered clips. **Design agreed with
   the owner (2026-11-26), not built yet:**
   - A **Video** (belongs to a Profile) is an ordered list of **shots**. The
     owner chooses how many shots (1–12) and **each shot's length** (3–15 s,
     fal's range); the total length and total price are shown. Each shot has its
     **own prompt** (motion/camera, English) — not one template for the whole
     video — and its own starting picture (make with AI / earlier clip / upload /
     link, as in the Generate dialog today). A single clip made from a template
     keeps working as it does now.
   - Plan vs attempts: a shot (`video_shots`: video, position, prompt, seconds,
     image) is the editable plan; every press of Generate on it is a normal
     generation row (`generations.shot_id`), so retakes keep their history and
     cost. The shot shows its latest result.
   - Spending: Generate per shot asks first with that shot's price (seconds ×
     the style's per-second price, $0.084 for 3D — linear scaling is NOT yet
     verified, check the first non-5 s bill); "Generate all unfinished shots"
     asks once with the total. The worker runs up to 5 at a time.
   - Continuity between shots: the profile's Look pictures plus the picture
     chosen for the previous shot go along as references when a keyframe is made.
     Taking the last frame of a clip needs ffmpeg (a new dependency: ask first),
     so it is not part of this.
   - The finished shots are downloaded (fal links last at least 7 days) and
     joined in CapCut for now (item 6).
   - Order of work, each a small PR from `main`: (1) tables + API (videos,
     shots, seconds on a generation, price per shot), (2) the Video page,
     (3) "generate all", continuity references, download list.
   - **(1) is built** (backend only; verified with the fake worker against
     Postgres, nothing spent): `internal/videos` — `GET/POST /videos`,
     `GET/PATCH/DELETE /videos/{id}`, `POST /videos/{id}/shots`,
     `PUT /videos/{id}/shots/order`, `PATCH/DELETE /videos/{id}/shots/{shotId}`,
     `POST /videos/{id}/shots/{shotId}/generate` (202; 409 if that shot is
     already being made, so a double click cannot pay twice; 422 without a prompt
     or a starting picture — a shot is always image-to-video, because that is the
     only path whose length and price are known). Deleting a video keeps the clips
     already made (they become ordinary clips); deleting a shot closes the gap in
     the order. `POST /generations` also takes `seconds` (3–15, default 5), the
     worker sends `generations.seconds` to fal, and `GET /studio/pricing` now
     returns the price per second (the screen multiplies by the length).
     Existing DBs need the two `CREATE TABLE`s (`videos`, `video_shots`) and
     `ALTER TABLE generations ADD COLUMN seconds smallint NOT NULL DEFAULT 5
     CHECK (seconds BETWEEN 3 AND 15), ADD COLUMN shot_id uuid REFERENCES
     video_shots (id) ON DELETE SET NULL;` (see `db/schema.sql`; create
     `video_shots` first).
   - **(2) is built** (the Videos tab, verified with the fake worker, nothing
     spent): list + "New video" (fal profiles only), the open video lives in the
     URL (`?tab=videos&video=<id>`). Each shot card has its prompt, a length
     (3–15 s, with the price), a starting picture (the same four sources as the
     Generate dialog, in `PictureModal`), Save, Generate (confirm with the price;
     it saves what is typed first, so the attempt matches the screen), the newest
     attempt with a video player, and move up/down/delete. Add a shot (max 12).
     The page polls every 5 s (20 min) while a shot is being made and refreshes
     the credit balance when it finishes. Clips made from a shot show as "Video
     shot" in the Generations tab.
   - **Why shots did not match, and the answer (owner's first real video,
     2026-11-26):** every shot is an independent clip from its own picture, and
     the pictures were made independently, so room, light and pose differed.
     Chaining the last frame of each clip into the next was considered and not
     chosen as the main method: quality drifts with every hop, the motion
     restarts from rest at each cut, it forces one-at-a-time generation, and
     grabbing frames needs ffmpeg (a new dependency: ask first). Short-form video
     does not need pixel continuity anyway: it is cut from short shots (3–4 s)
     that share a character, room and style. So:
     (a) **Same scene, other angle:** when making a shot's picture, up to 2
     pictures of the video's other shots can be sent first among the references
     (`sceneImageIds` on `POST /keyframes`, column `keyframes.extra_image_ids`),
     with an instruction to keep the room, props, lighting and character and
     change only what is described. The first other shot is ticked by default.
     The wording is a first guess; unverified how well either model holds a scene.
     (b) **Storyboard first:** the pictures of all shots side by side; each shot
     has an OK tick (`video_shots.approved`; changing the prompt, length or picture
     clears it; needs a prompt and a picture), and "Generate N approved shots"
     asks once with the total (`POST /videos/{id}/generate`, only approved shots
     are made, the rest come back as skipped with a reason; shots already made
     are left out by the screen). Pictures cost cents, clips ten times that, so
     the sequence is judged on pictures before paying for video.
     Existing DBs need `ALTER TABLE keyframes ADD COLUMN extra_image_ids uuid[]
     NOT NULL DEFAULT '{}';` and `ALTER TABLE video_shots ADD COLUMN approved
     boolean NOT NULL DEFAULT false;`.
   - **End picture (`end_image_url`), tried by the owner by hand against fal
     (2026-11-26):** with two pictures made independently (same cat and room,
     but the rug, a stool and the camera distance differed) the clip *morphed*
     one into the other: the rug shrank and the stool faded out. With an end
     picture made by editing the start picture ("exact same camera, framing,
     room, rug, lighting; only the pose changes") the owner found the clip
     continuous. So an end picture only works when the two pictures differ in
     the subject's pose and nothing else; the prompt should then say "static
     camera, no zoom" (the framing is already in the pictures).
     **Built:** each shot has an optional End picture (`video_shots.end_image_id`,
     a stored picture only, only with a starting picture; changing it clears the
     OK). "End picture…" opens the picture dialog in `end` mode: Make with AI
     starts from the shot's own starting picture with the strict
     `keepSceneSameCameraInstruction` (`sameCamera` on `POST /keyframes`), no
     link source; "Use next shot's" sets the end to the next shot's starting
     picture so a clip ends on the pose the next one starts from. The worker sends
     `end_image_url` (data URI) and `generations.end_image_id` records it. The
     confirm says the price with an end picture is not confirmed: **check the
     first bill** (a 4 s clip should be $0.336). Existing DBs need
     `ALTER TABLE video_shots ADD COLUMN end_image_id uuid REFERENCES images
     (id) ON DELETE SET NULL;` and the same column on `generations`.
     The picture the clip really ends on is close to, not identical to, the end
     picture. Chain pictures from the *first* picture when edits drift; use a
     cut (a new start picture of the same scene, no end picture) where the
     camera must change.
   - **Shot cards, chaining and takes (built after the owner's first edit):**
     "Use previous end" on a shot's starting picture sets it to the previous
     shot's end picture (the same stored picture serves as both, and is deleted
     only when nothing uses it). Every attempt at a shot is kept; the owner
     compares them in "Compare takes" and picks one (`video_shots.
     chosen_generation_id`, `PUT /videos/{id}/shots/{shotId}/take`, `GET …/attempts`;
     only a finished attempt of that shot can be chosen; empty goes back to the
     newest successful one; deleting the chosen attempt falls back to it). The
     shot shows its *take* (chosen, else newest successful) next to the state of
     the newest attempt, and "Clips for your editor" lists each shot's take in
     order with its link and "Copy all links". Existing DBs need
     `ALTER TABLE video_shots ADD COLUMN chosen_generation_id uuid REFERENCES
     generations (id) ON DELETE SET NULL;` (the two tables point at each other,
     so it is added after `generations`).
   - **Still to do in (3):** try Kling `elements` once for a character that
     changes inside a clip. Then item 5 (Thai voice-over) is the biggest lever
     for Shorts.
   - **Other video models on fal (read from fal's pages 2026-11-26, prices as
     listed):** Kling v3 Standard (ours) $0.084/s silent ($0.126/s with audio),
     3–15 s, has `end_image_url` and `elements` (1–3 reference pictures per
     element, `@Element1` in the prompt, a way to keep a character inside the
     clip), output shape follows the start picture (no aspect field); Kling v3
     Pro $0.112/s silent, "strong subject and text consistency"; Veo 3.1
     $0.20/s silent at 720p/1080p (Fast $0.10/s), 4/6/8 s, has 9:16, a
     first+last-frame endpoint and a reference-images endpoint (fixed 8 s);
     Seedance 2.5 about $0.47/s at 720p, 4–30 s, end image, no 9:16 field (follows
     the picture); Wan 3.0 Prime $0.14/s at 720p, 2–30 s, 9:16 and end image;
     MiniMax H3 and Seedance 2.0 Fast: price not stated. Kling Standard stays the
     best value at this size; Kling `elements` is the thing to try next for
     character consistency inside a clip. Not read: Luma Ray, Hailuo-branded
     endpoints, Kling v4.
4. Keyframe generation (image model on fal) instead of pasting URLs.
   **Built (done before item 3, decided with the owner), verified end to end
   with the fake generator; not yet run against real fal:**
   - Two image models, chosen per request in the Generate dialog ("Make with
     AI"): `nano-banana` (`fal-ai/nano-banana` / `…/edit`, $0.0398 per picture)
     and `seedream` (`fal-ai/bytedance/seedream/v4/text-to-image` / `…/edit`,
     $0.03). Prices are what fal's model pages list (2026-11-26), not yet seen on
     a bill. Registry: `internal/studio/keyframe.go`; the per-model request
     fields live in `internal/generate/image.go`, keyed by the model key.
   - The owner picks how many pictures (1–4) and sees the estimate first; the
     pictures are 9:16 (`studio.KeyframeAspect`).
   - **Reference pictures per Profile** (the "Look" tab, up to 4, JPG/PNG,
     ≤ 4 MB each; `profile_reference_images`) give a profile its own character.
     When it has any, the edit model is used and they are sent as data URIs,
     with "Keep the same character and overall look…" + the style's wording
     (`Recipe.KeyframeSuffix`) added to the prompt; a checkbox in the dialog turns
     them off. **Unverified:** that fal accepts four base64 pictures in one
     request, and how well either model keeps a character.
   - Candidates are downloaded by the worker and stored in `images` (a fal link
     expires). Picking one passes its `imageId` to the clip as before; the
     request and the unpicked candidates are deleted when the dialog closes
     (`DELETE /keyframes/{id}` keeps any picture a clip already uses). Deleting
     a profile now also removes the pictures only it used (this closes the old
     orphan-images gap).
   - A keyframe job retries at most twice (each retry is paid for); once the
     model has been called every outcome is recorded and the job is not retried.
   - Prompts: the image models' pages do not say whether Thai is supported;
     write picture descriptions in English until tried.
   - Existing DBs need the three `CREATE TABLE`s in `db/schema.sql`
     (`profile_reference_images`, `keyframes`, `keyframe_images`).
   - **Spending guards (owner's request):** every action that spends fal credits
     asks first with a price — "Make pictures" and fal "Generate" (a text-only
     profile is free and does not ask). The clip price comes from
     `GET /studio/pricing` (recipe `ImagePricePerSecondUSD` 0.084 × `ClipSeconds`
     5 = $0.42, from the first real clip); a text-to-video price is NOT known
     and the dialog says so.
   - "From an earlier clip" in the Generate dialog reuses a picture an earlier
     clip of the *same profile* used (no new picture to pay for; the UI lists
     only that profile's clips, the API does not otherwise check ownership).
   - **Credit balance in the menu bar:** `GET /credits/fal` reads
     `GET api.fal.ai/v1/account/billing?expand=credits`. Per fal's docs that needs
     the `billing:usage:read` permission (BILLING / READONLY / FULL preset), which
     an `API`-preset key is not documented to have, so it uses its own optional
     setting `falBillingKey`. Cached 30 s. **Not yet tried with a real key.**
   - **Real cost per clip (built 2026-11-27):** the worker saves fal's `request_id`
     on the generation (`generations.fal_request_id`, kept even when the clip failed
     after submit); `POST /credits/fal/costs` asks `GET api.fal.ai/v1/models/billing-events`
     (same `falBillingKey`; needs `billing:usage:read`, `auth:keys:read`,
     `account:users:read`) for those ids and stores `cost_total` in `generations.cost_usd`.
     The Generations tab shows a Cost column and a total. Billing can trail a clip by a
     few minutes, so unknown costs are asked again on the next load. Clips made before
     this have no request id and stay "—". **Verified:** the billing key is accepted by
     billing-events (a lookup of an unknown id returned 200, empty); **not yet seen
     with a real clip's bill.** Existing DBs need `ALTER TABLE generations ADD COLUMN
     fal_request_id text NOT NULL DEFAULT '', ADD COLUMN cost_usd double precision;`.
5. Thai voice-over (TTS provider chosen after the owner listens to Thai
   samples from each candidate). **Put on hold by the owner (2026-11-27);**
   it becomes more important again when the 2D educational style (item 7)
   is made, since such clips lean on narration.
6. Join clips + audio. **Conflicts with a rule above:** joining makes a full
   video file, and this project never stores full videos. Until the owner
   decides (allow it, or keep joining manual in CapCut), this stays manual and
   needs ffmpeg (a new dependency — ask first).
7. `2d` and `realistic` recipes. **The owner's 2D is flat vector cartoon for
   educational clips** (decided 2026-11-27). Expect: AI pictures with text in
   them are unreliable, so keep pictures free of text and add captions in the
   editor; Kling may add 3D-looking shading, so the wording needs "flat 2D
   vector, no shading, no 3D" and a real test (about $1–3); keep motion simple.
   `realistic` is for later and, for Shopee products, should start from the real
   product photo rather than an AI redraw.
8. **Script helper (built, owner-approved 2026-11-27; the Phase 2 "AI prompt
   suggestions" ban is lifted for this):** "Draft with AI" in the Videos tab.
   The owner gives a topic (Thai or English), how many shots (1–12) and how long
   each is; `POST /videos/draft` asks an LLM for a scene + per-shot picture
   prompt and motion prompt (JSON in the prompt, parsed and checked: wrong
   shape or too few shots is retried once, told what was wrong) and creates a
   Video with them filled in. It makes no pictures and no clips. The LLM runs
   through fal's `openrouter/router` with the SAME fal key (billed per token;
   the reply carries `usage.cost`, shown afterwards; no per-token rate is listed
   on fal's pages, so the price before the call is only "a few cents at most",
   which is an estimate). The model is `scriptModel` in Settings, default
   `anthropic/claude-sonnet-5` (an id fal's page lists as an example). **The
   owner ran a real draft on 2026-11-27 and it worked:** the id is accepted, the
   JSON parsed, and the draft was sensible, with one shared scene line. Its flaws
   were a style word and a look in a picture prompt ("a round-faced 3D cartoon
   cat") and an extreme close-up followed by a wide shot (no chaining possible);
   the system prompt now forbids both explicitly. The cost of that draft was not
   recorded here (the screen shows "It cost $X" after a draft).
   The system prompt (`internal/scriptwriter/prompt.go`) carries the studio's
   rules (motion-only clip prompts, one shared scene line, no text in pictures,
   nothing harmful to imitate, hook / payoff / loop) and the owner's cat-and-bag
   example. The scene line is put in front of every picture prompt. Shots now
   keep a `picture_prompt` (`video_shots.picture_prompt`; Make with AI starts
   from it and what is typed there is saved back); it is not sent to the clip
   model, so editing it does not clear the shot's OK. Existing DBs need
   `ALTER TABLE video_shots ADD COLUMN picture_prompt text NOT NULL DEFAULT '';`.
   **The draft also decides what chains** (owner's request, same day): per shot it
   may write an `endPicturePrompt` (`video_shots.end_picture_prompt`), only when
   the action carries straight on into the next shot: the end of the first
   picture's pose/position change, nothing else ("Only the …"), same camera, room
   and light. Then the NEXT shot starts from that end picture, so the next shot's
   picture prompt describes the state the end picture shows and its motion prompt
   begins from it; where the next shot cuts, the field is empty. The last shot
   never has one (the parser drops it). The "End picture…" dialog opens with that
   text, and what is typed there is saved back to the shot. The pictures are
   still made by the owner (no automatic spending); a "make all pictures" button
   was considered and left for later because the owner must pick among candidates
   and a bad picture spoils the ones edited from it. **Unverified:** how well the
   model writes end prompts (small enough changes) on real drafts.
   Existing DBs also need
   `ALTER TABLE video_shots ADD COLUMN end_picture_prompt text NOT NULL DEFAULT '';`.
   **Thai prompts (built 2026-11-27, the owner reads English poorly).** The owner
   writes and edits prompts in Thai; the English is what the models receive (fal's
   pages do not promise Thai for the picture or video models). Each shot keeps the
   three prompts twice: `video_shots.prompt_th` / `picture_prompt_th` /
   `end_picture_prompt_th` beside the English columns. `POST /videos/translate`
   `{kind: motion|picture|endPicture, direction: toEnglish|toThai, text}` →
   `{text, costUsd, model}`, through the same fal LLM route as the draft, with the
   rules of each kind (a clip prompt says only what moves; a picture prompt has no
   style words, text or looks; an end picture prompt starts "Only the …", a pose
   change only; always "translate, add nothing"). The English answer is checked to
   hold no Thai letters (retried once if it does; both costs are added). The model
   is `translateModel` in Settings, else `scriptModel`, else the default. Draft with
   AI also writes a Thai twin of every prompt (the scene's Thai goes in front of the
   Thai picture prompt, as the English does). **Screen:** each prompt is a Thai box
   with "แปลเป็นอังกฤษ" / "ดูความหมายเป็นไทย" and the English folded under it with a
   preview, never hidden. The English counts as made from the Thai when the page
   loads; editing the Thai marks it out of date, and Save, Generate, the picture
   dialogs and "Make pictures" translate any out-of-date Thai first. The Make
   pictures confirmation and the Generate confirmation show the English that will be
   sent. A Thai letter in an English box blocks Generate. The cost of a translation
   is shown after it ("แปลแล้ว ≈ $…"); there is no confirmation for it (a fraction of
   a cent, owner's choice 2026-11-27). **Not counted** in the daily spending limit or
   the dashboard's spend (neither the draft nor translation is billed per clip); fal
   credits still drop. Existing DBs need
   `ALTER TABLE video_shots ADD COLUMN prompt_th text NOT NULL DEFAULT '', ADD COLUMN
   picture_prompt_th text NOT NULL DEFAULT '', ADD COLUMN end_picture_prompt_th text
   NOT NULL DEFAULT '';`. **Unverified against real fal:** how faithful the
   translations are (the owner can check with "ดูความหมายเป็นไทย").
9. ~~Video templates~~ **cut by the owner (2026-11-27):** not needed now.

## Realistic style and the Cast (built 2026-11-27)

The owner's `realistic` style has two uses, built one at a time: **product clips** first (a Shopee
product that must look like the real one, in every shot — owner's choice, riskier than ending on the
real photo; to be measured), then **short films** (≈30 s, satirical humour, visual gags with Thai
captions added in the editor while voice-over is on hold).

- **`realistic` recipe** (`studio.StyleRealistic`): pictures get "Photorealistic photograph, natural light,
  realistic materials and skin…" (never "cinematic"), `ImageSuffix` is empty so a real photo is never
  restyled. Wording is a first guess, not tried on real output.
- **The Look tab became the Cast tab.** `cast_members` (role `hero|costar|prop|place`, name, note,
  position; max 8 per profile) + `cast_member_images` (1–4 pictures per member) replace
  `profile_reference_images`. API: `GET/POST /profiles/{id}/cast`, `PATCH/DELETE …/cast/{castId}`,
  `POST …/cast/{castId}/images`, `DELETE …/images/{imageId}`. A profile's hero is one recurring
  person/character; co-stars, props and places are optional. **Product pictures do NOT live here** — the
  decision is that they belong to the catalog `Product` (photos with a state label: closed / open / in
  hand), not built yet.
- **Picture requests name the pictures.** `POST /keyframes` takes `castIds` (replacing `useReferences`):
  the ticked members' pictures are stored on the request (`keyframes.cast_image_ids`, after the scene
  pictures) and `studio.KeyframePrompt` says which is which ("Pictures 1-2 are "Name", the lead
  character (note). Picture 3 is …") then "each must look exactly as in its own picture(s)…". The
  dialog ticks the hero by default. One request carries at most 6 pictures in all (scene + cast,
  `studio.MaxRequestPictures`), a cautious cap. **First real result (owner, 2026-11-28):** one request
  with 2 reference pictures (hero "พี่นักแบด" + prop "กระติกเขียว", a green bottle) came back as the hero
  in a red jersey drinking from the green bottle on a badminton court; the owner judged it about right.
  So fal accepts two base64 reference pictures in one request and the naming prompt works at that size.
  **Still unverified:** 3–6 pictures, the bottle's logo/text, how steadily the face matches across
  shots (a single sample), and which model did it (not recorded).
- **Lessons from the owner's first bottle test:** an edit model copies the reference's *state* too — a
  closed bottle in the reference stays closed whatever the prompt says, so the pictures given must show
  the state the shot needs (open cap, in hand). One colour per request. A photo whose ratio is under 0.4
  (e.g. 311×788) is rejected by `images.Check`; pad it.
- Existing DBs need the migration: `CREATE TABLE cast_members`, `cast_member_images` (see
  `db/schema.sql`), `ALTER TABLE keyframes ADD COLUMN cast_image_ids uuid[] NOT NULL DEFAULT '{}';`, then
  `INSERT INTO cast_members (profile_id, role, name, position) SELECT DISTINCT profile_id, 'hero',
  'Look', 1 FROM profile_reference_images;` and `INSERT INTO cast_member_images (cast_id, image_id,
  position) SELECT c.id, r.image_id, r.position FROM profile_reference_images r JOIN cast_members c ON
  c.profile_id = r.profile_id AND c.name = 'Look';` then `DROP TABLE profile_reference_images;`. The worker
  must be restarted with the api (it reads `cast_image_ids`).
- **Kling `elements` (read from fal's pages and OpenAPI, 2026-11-27; not tried):** our endpoint
  `fal-ai/kling-video/v3/standard/image-to-video` already takes an optional `elements` list, no new
  model. Each element is `{frontal_image_url, reference_image_urls (1–3, at least one is required),
  video_url, voice_id}`; the prompt refers to them as `@Element1`, `@Element2`; `start_image_url` stays
  required and `end_image_url` still works; images 50 MB, ≥ 300×300, ratio 0.4–2.5. Price is listed as
  the same $0.084/s silent (0.126 with audio, 0.154 with voice control) — **check the first bill for an
  extra charge**. A request may hold only one element with a video. **Not stated:** a maximum element
  count, whether base64 data URIs are accepted for element pictures (they work for the start picture),
  and whether elements hold a *product's* logo as well as a face (the page says "characters/objects").
  `generate_audio` knows only Chinese and English (other languages are translated to English), so the
  silent-clip + Thai-TTS decision stands. A sibling endpoint `…/o3/standard/reference-to-video` takes
  `image_urls` (`@Image1`), an optional start picture and `aspect_ratio` incl. 9:16, same $0.084/s
  silent, `generate_audio` default false; its combined cap is stated only as "Maximum 4 total (elements
  + reference images)" when a video is used. Fit with the cast: a member with 2–4 pictures maps to one
  element (first = frontal, the rest = references; a member with one picture cannot be an element
  unless fal accepts the same picture twice). Elements would help when the subject turns or moves in a
  clip and would keep the same person across shots without chaining frames. Plan: one A/B pair of 5 s
  clips from the same start picture, with and without elements (≈ $0.84), before any code.
- **Product pictures (built 2026-11-28):** `product_images` (product, image, position 1–4, `label`;
  removed with the product) — the real product from several angles and STATES (cap closed / open /
  in hand), because an edit model copies the state of its reference. Products page: a "Pictures"
  column opens `ProductPicturesModal` (upload with a label, edit label, remove). API:
  `GET/POST /products/{id}/pictures` (multipart `file` + `label`, JPG/PNG ≤ 4 MB), `PATCH/DELETE
  …/pictures/{imageId}`, `GET /products/pictures` (products that have pictures, for the dialog).
  `POST /keyframes` takes `productPictureIds` (image ids; at most 2 different products, counted in
  the 6-picture cap): they follow the cast's pictures and the prompt names each one — `Picture 4 is
  "Bottle", the product (cap open).` (`studio.RoleProduct`, not a valid cast role). The picture dialog
  lists each product's pictures with ticks (none ticked by default). Existing DBs need `CREATE TABLE
  product_images …` (see `db/schema.sql`). **Verified:** build, unit tests for the prompt wording,
  the dialog loads a product's (empty) list against Postgres. **Not verified live:** upload, label
  edit, remove, the product-picture checks on `POST /keyframes`, and any real picture made from them.
- **Draft with AI knows the format, the style and the cast (built 2026-11-28):** the dialog has a
  Format switch — General (as before), **Product** (needs a catalog product that has pictures) and
  **Film** (satirical short told in visual gags, no dialogue; Thai captions are added in the editor) —
  sent as `mode` and `productId` on `POST /videos/draft`. `scriptwriter.systemFor` appends to the standing
  prompt: the mode's structure and rules (`modes.go`: product = hook → natural use → benefit through
  action → calm static product shot, show the product ONLY in the states its pictures have, no label
  close-ups, no hands turning it, no claims beyond the owner's note on the product; film = setup →
  escalation → punchline → last beat, one place, one main character), a photographic block when the
  profile's style is `realistic`, and, when the profile has a cast, how to refer to it: **in the English
  fields only by role words** ("the lead character", a plain generic noun for co-stars and props, "the
  location") — never the names, because names are often Thai and an English box with Thai blocks
  Generate, and because the picture step names each picture with the same role words; the Thai twins use
  the names. The user prompt lists the cast and the product (name, the owner's note, the labels of its
  pictures). The writer sees only text; which cast members and product pictures are in a shot is still
  ticked by the owner when the picture is made. A test keeps `the lead character` identical on both
  sides. **Not tried on a real model yet:** whether the model follows the product rules (states,
  no logo close-ups) and writes a joke that lands, and what a draft costs.
- **First short-film test (owner, 2026-11-28; plot: another character drinks from plastic bottles, a cut
  to a wider shot shows three empties, a cut to the hero drinking from the green bottle, cap open) —
  what a real Film draft got wrong, to fix in the system prompt and a linter:**
  1. the scene line carried camera framing ("A wide view … showing the full court") and a style phrase;
     it must say only the place (and it is copied in front of EVERY shot's picture prompt, so a fix means
     editing every shot — storing the scene once on the video is a to-do);
  2. role words ("costar", "lead character") in clip (motion) prompts — the clip model sees only the
     start picture and the text, so use a plain noun ("the person", "the player"); role words belong only
     in picture prompts, where the picture step names the reference pictures with the same words;
  3. camera instructions that contradict each other ("pulls back" + "Static camera, slight slow zoom
     out"): the standing rule makes the model append "Static camera, no zoom" even when it asked for a move;
  4. the start picture already shows the action the motion begins ("tilting his head back to drink" vs
     "raises the bottle") — the picture must be the moment BEFORE the motion;
  5. negatives in picture prompts ("nothing visible around his feet"): frame instead ("from the chest
     up") — and a zoom-out that reveals what is not in the first frame makes the model invent it, so
     do the reveal as a cut to a wider picture;
  6. static objects mentioned in a motion prompt (the empty bottles), three beats in 5 s, mood words
     where a concrete action reads better (a small smile and a slow exhale);
  7. "badminton player" written for a character who is only sitting (the model may add a kit/racket);
  8. cast notes are sent as typed inside the picture prompt: write them in English ("cap open"), names may
     stay Thai; image models do not promise Thai. Object counts ("exactly three") are unreliable: make
     3–4 candidates and pick one.
  **Verified by the owner:** ticking "Keep the same scene as" an earlier shot's picture (which holds
  another character) did NOT pull that character into the hero's picture (one sample).
  Also found: the profile description limit counts bytes, not characters (Thai ≈ 200 characters of 600
  bytes) while its message says characters.
- **Next** (agreed order): ~~product pictures on `Product`~~ (done above); ~~Draft knows the modes~~
  (done above); then Kling `elements` (A/B test of two clips first, ≈ $0.84) and the `2d` style. + a "product" role in the picture dialog; the
  script helper learning the product mode and the cast names; a real test (hero + product, ≈ $0.16 per
  picture model); Kling `elements` research for keeping a person/object inside a clip.

## Video models (built 2026-11-27)

A Profile now picks its **video model** (`profiles.video_model`: `kling-v3` default, `veo-3.1`, `veo-3.1-fast`) and
whether clips have **sound** (`profiles.video_audio`, off by default; a switch on the profile form). Registry:
`internal/studio/models.go` (fal ids, allowed lengths, default length, price per second, whether text-only clips and an
end picture are possible). `internal/generate/fal.go` builds the input per model family: Kling as before; **Veo**
(read from fal's pages 2026-11-27): `image_url`, `duration` "4s"/"6s"/"8s", `aspect_ratio` "9:16", `resolution` "720p",
`generate_audio`; an end picture needs the separate `first-last-frame-to-video` endpoint (`first_frame_url` +
`last_frame_url`). Veo takes only 4, 6 or 8 s, so lengths are checked against the profile's model when a clip or shot is
made (a shot kept from before a switch says what to change), the length box becomes a list for Veo, and
`GET /studio/pricing` now returns `{models:[…]}` (per model: lengths, default, USD/s silent and with sound, whether
the silent / with-sound price was confirmed on a bill). **Verified with a real clip (2026-11-27):** one Veo 3.1 clip, 6 s,
sound ON, was billed **$2.40** (= $0.40/s, so Veo with sound is twice the silent $0.20/s); the request shape worked. The clip
did NOT follow the prompt: the starting picture already had the bag on the floor and the cat looking at the camera, while the
prompt said "the bag drops" and "eyes on the bag", so Veo kept the picture's pose (cat looks at camera), then improvised
(looked down, turned, pushed the bag out of frame). A second try whose prompt began from what the picture shows ("the cat,
staring at the camera, slowly turns its head down to the bag; ears tilt forward; the bag does not move") was followed.
**Write a clip prompt as the next move from the picture, never as something the picture already contradicts.** The
owner kept Veo as a second option for its speed (faster than Kling's ~8 min) and for sound without narration; Kling stays
the default for controlled motion and is ~5x cheaper. Veo Fast offers only image-to-video (its text and first-last endpoints
were not read) and its with-sound price is not seen on a bill. Existing DBs need `ALTER TABLE profiles ADD COLUMN video_model text NOT NULL DEFAULT 'kling-v3',
ADD COLUMN video_audio boolean NOT NULL DEFAULT false;`.

## Prompt rules and a prompt linter (built 2026-11-28)

After the owner's first short-film test (the mistakes are listed with the Draft modes in the Realistic /
Cast section, once that PR is merged). The standing prompt (`scriptwriter/prompt.go`) now says: the scene is
the PLACE only (no framing, no style phrase); motion prompts use plain nouns and never role words ("hero",
"costar", "lead character"); end with ONE camera sentence ("Static camera, no zoom." by default; one slow
movement only when asked, never both, and never a move that reveals what is outside the start picture — cut
to a wider picture instead); the picture is the moment BEFORE the action; no negatives in picture prompts;
a feeling is a visible action. The worked example no longer combines "static camera" with a zoom.

`lib/promptLint.ts` runs on the English of every prompt as it is typed (shot cards and the picture dialog)
and shows ⚠ lines, with a one-click fix for role words ("the person"): role words in motion, static camera +
a movement, "reveal", a framing phrase at the start of a picture prompt ("A wide view…", "showing the
full…"), negatives in a picture prompt, an end picture not starting "Only". It only warns, and skips a
prompt whose Thai is ahead of its English. `scripts/promptLint.smoke.mjs` checks it on the owner's real draft
texts (no test framework is installed: `node --experimental-strip-types scripts/promptLint.smoke.mjs`).
**Not detected** (needs the model): a start picture that already shows the action, and too many beats.
Profile name / niche / description limits now count characters, not bytes (Thai is 3 bytes a letter).

## Price check, daily limit and the rendering indicator (built 2026-11-27)

- **Live prices.** `GET /studio/pricing` also asks fal's pricing API (`GET api.fal.ai/v1/models/pricing?endpoint_id=…`,
  any API key, no billing permission; cached 10 min, 1 min after a failure) and adds `liveUsdPerSecond` /
  `priceChanged` per model. **What the number is (probed 2026-11-27):** fal answers ONE `unit_price` per endpoint, unit
  "seconds", with no split by option, and it is the top of the range: Veo 3.1 0.40 (with sound; silent 0.20), Veo Fast
  0.15, Kling 0.14 (silent 0.084 on a real bill). So it is a **ceiling**, not the price of a silent clip, and it
  cannot confirm silent prices. The table in `studio/models.go` holds the with-sound price (Kling 0.14, Veo 0.40, Fast
  0.15; an earlier read said Kling 0.126); `priceChanged` is true when fal's number differs from it. The confirm
  dialogs then add a warning, and with sound they use fal's live number. Real cost still arrives after the clip
  (billing-events, Cost column); nothing yet compares estimate to bill automatically.
- **Daily spending limit.** Settings has "Daily spending limit" (`dailyCapUsd`, empty = none) and the browser's time
  zone is saved with it (`dailyCapTz`) so the day starts at the owner's midnight. `internal/spendcap` is checked before
  anything is saved or queued: a clip from a template (`POST /generations`), a shot (`queueShot`, so "generate all"
  counts the shots already queued), and a picture request (`POST /keyframes`). Over the limit → **402** with the
  message. Spent today = the real bill where known, else the model's list price (clips still rendering or not billed
  yet; a failed clip with no bill counts 0; pictures at the image model's list price). If the guard cannot read its
  numbers it lets the spend through and logs it. Verified live: a 1-cent limit refused a picture request with 402 and
  created nothing. Not covered: the script helper (a few cents) and anything started outside the API.
- **Rendering indicator.** The menu bar shows a pill "N rendering" only while something is being made; click opens a
  popover (a bottom sheet on a phone) listing each clip and picture request with what it is, profile, model, length
  and a live elapsed time, linking to the profile. Source: `GET /dashboard/running` (clips + picture requests with
  status pending), polled every 5 s while busy, 20 s idle, and at once after any POST to /generations, /videos or
  /keyframes; the credit balance refreshes when the list shrinks.
- The sun/moon theme switch's chosen item is now a circle (antd rounds it less than the track by default).
- **Status tag (`components/Tally.tsx`):** a glowing dot in a capsule tinted with the status colour (`--tally`); Rendering
  still pulses, Done and Failed are still. Stronger capsule/glow in light mode.
- **Profiles page:** a List / Cards switch and a sort (Recently updated, Newest first, Name A–Z, Rendering first), both
  remembered in the browser (`localStorage` `acmp.profiles.view` / `.sort`). A profile with clips rendering shows a pulsing
  "N rendering" tag that links to its Generations tab (from the shared poll `app/useRunning.ts`; picture requests are not
  counted). The card has a cover for the profile's picture: profiles have no picture yet, so it shows the tape stripe;
  `Cover` in `app/profiles/page.tsx` already takes an `imageUrl` (upload + a `profiles` column are still to do).
  A profile's tabs are now Generations, Videos, Prompt templates, Look, Channels, and Generations is the default.
- No schema change for any of this.

## Dashboard (built 2026-11-27)

Two screens read `GET /dashboard?tz=<IANA zone>` (`internal/dashboard`, queries in `db/query/dashboard.sql`):
**Home** (`/`, the page after sign-in — credits left, spent today, clips rendering, recent failures; the four
getting-started steps show instead until a profile exists) and **Dashboard** (`/dashboard`, its own menu entry: the
full picture below). On a phone the tab bar is Home, Dashboard, Profiles, Products, Account (Settings is in the
Account sheet). One reply holds: spend (this month / all time /
per day for 30 days / per profile, from `generations.cost_usd`; clips with no bill are counted apart, never as $0),
clips rendering and the latest failures, production counts (videos, shots OK'd / made, finished clips not
published), YouTube uploads (7 d / 30 d / total / latest) and Shopee products (pinned or not). Day and month
boundaries follow the browser's time zone. It only reads existing tables; no schema change. **Not built (slots are
shown as "Not connected"):** YouTube views/likes (the Data API `videos.list` statistics part may work with the
`youtube.readonly` scope already granted — to be tried; watch time and revenue need `yt-analytics.readonly`, a new
consent) and Shopee clicks/orders/commission (Phase 3 B).

## Picture zoom and small hints (built 2026-11-28)

AI pictures were too small to judge. `components/ZoomableImage.tsx` wraps antd's `Image` preview (full
screen, zoom, Esc to close; no new dependency): a shot's starting and end picture, cast pictures and
product pictures open large when clicked. In the picture dialog a candidate is picked by clicking it, so
it keeps that click and has its own "ขยายดู" link (`ZoomLink`) under it; candidates are also bigger
(140×249). Hints: cast notes and product picture labels should be written in English ("cap open"), since
they are sent to the image model as typed (names may stay Thai); when the English picture prompt holds a
count ("exactly three bottles") and fewer than 3 pictures are asked for, a hint suggests 3–4 because image
models often get counts wrong. **Verified on screen:** clicking a cast picture opens it large. **Not
looked at on screen:** the "ขยายดู" link and the count hint (they need candidate pictures; type-checked only).

## Thai voice samples (built 2026-11-28)

The voice-over plan is in the 2D / Thai voice section (research, candidates, order of work). This is its
step "listen before choosing": Settings → **Thai voice samples**. The owner keeps or replaces a Thai text (a
~300-character educational paragraph is the default), ticks candidate voices, confirms the price, and each
voice speaks the SAME text, with an audio player per voice. Nothing is saved (fal keeps each file at least
7 days); a voice that fails (ElevenLabs may refuse Thai) shows its own error and costs nothing.
- Registry `studio/voices.go` (`VoiceCandidates`): MiniMax Speech 2.8 HD ($0.10 per 1K characters) with the
  three Thai system voices from MiniMax's own list (`Thai_female_1_sample1` "Confident Woman",
  `Thai_male_1_sample8` "Serene Man", `Thai_female_2_sample2` "Energetic Woman"), MiniMax 2.8 Turbo ($0.06)
  with two of them, ElevenLabs v3 and Multilingual v2 (default voice Rachel, `language_code` "th"; $0.10).
  A 400-character text on four voices is a few cents.
- `generate.FalVoice` (`generate/voice.go`) builds the body per family: MiniMax 2.8 takes the text in
  `prompt`, `language_boost: "Thai"`, `output_format: "url"` (the default is hex) and
  `voice_setting.voice_id`; ElevenLabs takes `text`, `voice`, `language_code`. The reply holds `audio.url`
  and, for MiniMax only, `duration_ms`. `FakeVoice` (AI_FAKE_FAL=1) returns a second of silence as a data URI.
- API: `GET /voice/candidates` (voices, prices, limits), `POST /voice/samples` `{text, candidates[]}` →
  `{results[{key,label,ok,audioUrl,durationMs,costUsd,error}], chars, totalCostUsd}`; 20–1500 characters, at
  most 12 voices (with `customVoices`); the daily spending limit is checked first (402), the voices run at once.
- **Heard by the owner (2026-11-28, same Thai text):** **MiniMax 2.8 misread Thai a lot** (HD and Turbo,
  the Thai system voices); **ElevenLabs v3 read it without a mistake, accent acceptable** (voice Rachel,
  `language_code: "th"` accepted on fal, no duration in the reply). The ElevenLabs **Multilingual v2 model
  does not support Thai**; the voices **Aria and Brian read Thai but not well** (first reported the other way
  round, corrected by the owner), and all three were removed. After hearing every voice the owner KEPT four:
  **Rachel, Sarah, Roger and Chris** (ElevenLabs v3; Charlotte, Daniel and George were dropped). MiniMax stays supported in `generate/voice.go` but is
  not offered. **Chosen model: ElevenLabs v3** ($0.10 per 1K characters). The four voices are the choices a narrator is picked from (likely per profile, to decide).
- **Choosing among voices that differ:** ElevenLabs' official per-voice descriptions (gender, age, accent,
  tone) were NOT reachable (help centre 403, voice library is script-rendered), so none is invented here; the
  authenticated `GET /v2/voices?category=premade` would return them. The screen instead lets the owner
  (1) try any other voice by typing a name or an ElevenLabs voice id, or clicking one of fal's example names
  (`studio.CustomVoice`: letters, digits, space, `_`, `-`; spoken in v3; up to 12 voices per request),
  and (2) write a note under each voice they heard (kept in the browser's localStorage, key
  `acmp.voiceNotes`).
- **Deadline:** ElevenLabs' docs say its **Default voices (the names fal lists: Rachel, Sarah, Roger…) expire
  on 2026-12-31** and are replaced by "Infinity" voices (marked ∞ in the Voice Library, "an infinite notice
  period"). Those docs give no Infinity names or ids, no API or Thai statement for them, and fal's `voice`
  field page says nothing about ids or library voices. So a narrator chosen among the Default voices works
  only until then; **before building on one, test an Infinity / library voice id through the custom field**
  (and keep the narrator as a setting, not code, so it can be swapped). Voice Library voices are not
  available to ElevenLabs free-tier API users.
  Samples are not counted in the daily spend or the dashboard (like the Draft and translations), though fal
  credits drop.

## Narration and the voice-over of a shot (built 2026-11-28)

For clips with a narrator (the 2D educational ones): the clip is silent, the Thai narration is spoken by a TTS
and joined in the editor. Order: write the narration → speak it → set the shot length from the audio → pictures
→ clips (a clip's length fixes its price, so the audio comes first; a picture does not depend on it).
- `video_shots.narration_th` (Thai, ≤ 1500 characters): what the narrator says over the shot. Plain Thai, no
  English twin (ElevenLabs reads Thai directly); never sent to a picture or video model, so it never affects the
  OK (approval). Saved with the rest of the plan (Save, or when the voice is made).
- `shot_audio` (one row per shot, replaced when spoken again): the MP3 bytes in Postgres like pictures (the fal
  link expires after at least 7 days; move to R2 when it exists), the voice key, the exact text spoken and
  `duration_ms`. **The duration is approximate**: fal's ElevenLabs reply carries none, so it is worked out from the
  size of the 128 kbps CONSTANT-bitrate MP3 we ask for (`output_format: mp3_44100_128`,
  `generate.ApproxMP3DurationMs`, about ±0.1 s). MiniMax would return a real `duration_ms` (used when present).
- **Narrator voice = a global setting** `narratorVoice` (a key from `studio/voices.go` or `custom:<name>`),
  chosen with "ใช้เป็นเสียงพากย์" under a voice in Settings → Thai voice samples. It is global for now because
  the per-profile choice needs the Profile columns of the 2D PR (look notes); a per-profile narrator is the next
  small step. ElevenLabs' Default voices expire 2026-12-31 (see above), so the narrator is a setting, not code.
- API: `POST /videos/{id}/shots/{shotId}/voice` (speaks the saved narration with the narrator; 422 without
  narration or narrator; the daily limit is checked first, 402; returns the shot), `GET …/voice/audio` (the MP3),
  `DELETE …/voice`. The shot DTO has `narrationTh` and `voice {voice, text, durationMs, sizeBytes, createdAt,
  current}` (`current` = the narration is unchanged since the audio was made).
- Screen (shot card): a Thai narration box with the characters and an ESTIMATE of the time
  (`lib/narration.ts`, `SPEECH_CHARS_PER_SECOND = 11`, a guess to calibrate on real audio) and a warning when it
  will not fit the shot; "สร้างเสียง" (confirms with the price, saves the plan first, replaces the old audio); a
  player with the real length; "ตั้งความยาวช็อตเป็น N วินาที ตามเสียง" (the shortest length the video model takes that
  holds the audio plus 0.5 s; not saved until Save); "ลบเสียง". `scripts/narration.smoke.mjs` checks the helpers.
- Existing DBs need `ALTER TABLE video_shots ADD COLUMN narration_th text NOT NULL DEFAULT '';` and `CREATE TABLE
  shot_audio …` (see `db/schema.sql`). Restart the api.
- **Not done / unverified:** a real voice made from a narration (the first one also tests that fal returns an
  https MP3 link that `FetchAudio` can read, and that the 128 kbps duration is right), the characters-per-second
  guess, Draft writing the narration (an "Explainer" format is the next step), an audio link in "Clips for your
  editor", and voices longer than ~1500 characters. Voice-over costs are not in the daily spend or the dashboard
  (like the Draft and translations), though the daily limit is checked before speaking.

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
