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
9. ~~Video templates~~ **cut by the owner (2026-11-27):** not needed now.

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
