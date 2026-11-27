-- Database schema. This file is the single source of truth for the DDL:
--   * docker-compose mounts it as a Postgres init script for local dev
--   * sqlc reads it to type-check queries and generate Go models
--
-- When a real migration tool is added later, its first migration should match
-- this file exactly.

CREATE TABLE IF NOT EXISTS profiles (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text        NOT NULL,
    niche       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    -- Which Generator this profile defaults to (see internal/generate).
    -- Stores the setting only — nothing reads it automatically yet.
    provider    text        NOT NULL DEFAULT 'text',
    -- Which video style its clips use ('3d', later '2d', 'realistic'). The name
    -- only: what a style means lives in internal/studio. Checked by the API, not
    -- here, so adding a style needs no schema change. Used by fal profiles.
    video_style text        NOT NULL DEFAULT '3d',
    -- Which video model makes its clips ('kling-v3', 'veo-3.1', ...). The name
    -- only: ids, lengths and prices live in internal/studio (models.go). Checked
    -- by the API, not here. Used by fal profiles.
    video_model text        NOT NULL DEFAULT 'kling-v3',
    -- Ask the model for sound too. Off by default: Thai narration is added
    -- separately, and sound costs more.
    video_audio boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- A channel is one distribution target under a profile (a YouTube channel, a
-- TikTok account, ...). Phase 1 only stores the record — no platform APIs yet.
-- Deleting a profile removes its channels.
CREATE TABLE IF NOT EXISTS channels (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    name        text        NOT NULL,
    platform    text        NOT NULL,
    handle      text        NOT NULL DEFAULT '',
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS channels_profile_id_idx ON channels (profile_id);

-- Reusable prompt templates associated with a profile. `body` is the template
-- text; Phase 1 stores it verbatim (no variable substitution yet).
CREATE TABLE IF NOT EXISTS prompt_templates (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    name        text        NOT NULL,
    body        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS prompt_templates_profile_id_idx ON prompt_templates (profile_id);

-- A picture the owner uploaded from their computer, used as the starting image
-- of an image-to-video clip. Kept in Postgres for now (small JPG/PNG, capped
-- by the API); move to object storage when R2 exists. Never video.
CREATE TABLE IF NOT EXISTS images (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    content_type text        NOT NULL,
    size_bytes   integer     NOT NULL,
    data         bytea       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- A profile's cast: the recurring characters, objects and places its pictures
-- keep the same (the "Look" tab). Each member has a role, a name (used in the
-- prompt so the image model knows which picture is who), an optional note
-- ("red kit", "cap open") and 1-4 pictures of itself. The owner ticks which
-- members appear when a picture is made. Removed with the profile. (This
-- replaced profile_reference_images, where the pictures were one flat list.)
CREATE TABLE IF NOT EXISTS cast_members (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    -- hero = the lead character, costar = another character (person, animal),
    -- prop = an object, place = a location.
    role       text        NOT NULL CHECK (role IN ('hero', 'costar', 'prop', 'place')),
    name       text        NOT NULL,
    note       text        NOT NULL DEFAULT '',
    position   smallint    NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cast_members_profile_id_idx ON cast_members (profile_id);

CREATE TABLE IF NOT EXISTS cast_member_images (
    cast_id  uuid     NOT NULL REFERENCES cast_members (id) ON DELETE CASCADE,
    image_id uuid     NOT NULL REFERENCES images (id) ON DELETE CASCADE,
    position smallint NOT NULL CHECK (position BETWEEN 1 AND 4),
    PRIMARY KEY (cast_id, position),
    UNIQUE (cast_id, image_id)
);

-- One request to make keyframe pictures (the starting image of a clip) with an
-- image model. Like a generation it is created pending and finished by the
-- worker; the pictures it made are in keyframe_images. `prompt` is exactly what
-- was sent (the profile's style wording already added). Removed with its profile.
CREATE TABLE IF NOT EXISTS keyframes (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    prompt     text        NOT NULL,
    -- Key of the image model chosen (internal/studio), e.g. 'nano-banana'.
    model      text        NOT NULL,
    -- How many pictures were asked for, and how many reference pictures went
    -- with the request (0 = made from the text alone).
    count      smallint    NOT NULL,
    reference_count smallint NOT NULL DEFAULT 0,
    -- Pictures of the SAME scene to keep (the other shots of a video), sent
    -- before the profile's reference pictures; reference_count includes them.
    -- Not a foreign key: a picture deleted meanwhile is just skipped.
    extra_image_ids uuid[] NOT NULL DEFAULT '{}',
    -- The cast members' pictures that went with the request (the ones the owner
    -- ticked), sent after the scene pictures, in this order. Stored so the worker
    -- sends exactly what the prompt describes even if the cast changes meanwhile.
    -- Not a foreign key, like extra_image_ids. reference_count includes them.
    cast_image_ids uuid[] NOT NULL DEFAULT '{}',
    status     text        NOT NULL,
    error      text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS keyframes_profile_id_idx ON keyframes (profile_id);

CREATE TABLE IF NOT EXISTS keyframe_images (
    keyframe_id uuid     NOT NULL REFERENCES keyframes (id) ON DELETE CASCADE,
    image_id    uuid     NOT NULL REFERENCES images (id) ON DELETE CASCADE,
    position    smallint NOT NULL,
    PRIMARY KEY (keyframe_id, position)
);

CREATE INDEX IF NOT EXISTS keyframe_images_image_id_idx ON keyframe_images (image_id);

-- A video: an ordered list of shots under a profile (Phase 4 item 3). The
-- owner chooses how many shots and how long each is. Removed with its profile.
CREATE TABLE IF NOT EXISTS videos (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    title      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS videos_profile_id_idx ON videos (profile_id);

-- One shot of a video: the editable PLAN (prompt, length, starting picture).
-- Every press of Generate on it makes a normal row in `generations` (see
-- generations.shot_id), so retakes keep their history and cost. At most 12 per
-- video and 3-15 seconds each (fal's range). The position is unique per video;
-- the constraint is deferred so a reorder can swap positions inside one
-- transaction.
CREATE TABLE IF NOT EXISTS video_shots (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id   uuid        NOT NULL REFERENCES videos (id) ON DELETE CASCADE,
    position   smallint    NOT NULL CHECK (position BETWEEN 1 AND 12),
    prompt     text        NOT NULL DEFAULT '',
    -- What the starting PICTURE should show (for Make with AI), kept with the plan
    -- so a drafted shot arrives with it filled in and it is not lost when the
    -- dialog closes. Not sent to the clip model, so changing it does not clear
    -- the approval.
    picture_prompt text    NOT NULL DEFAULT '',
    -- What changes between the starting picture and the END picture (only the
    -- subject's pose or position: the camera and room stay exactly the same).
    -- '' for a shot that has no end picture. Kept so a drafted shot arrives with
    -- it; the same note about the approval applies as for picture_prompt.
    end_picture_prompt text NOT NULL DEFAULT '',
    -- The owner's Thai versions of the three prompts above. The owner writes and
    -- edits these; the English ones are what the models receive (their pages do
    -- not promise Thai), made from the Thai by the translation helper. Kept so
    -- the owner can always read and change what a shot says in Thai. Not sent to
    -- any model, so they never affect the approval.
    prompt_th text NOT NULL DEFAULT '',
    picture_prompt_th text NOT NULL DEFAULT '',
    end_picture_prompt_th text NOT NULL DEFAULT '',
    seconds    smallint    NOT NULL DEFAULT 5 CHECK (seconds BETWEEN 3 AND 15),
    -- The picture the shot starts from: a link, or an uploaded/made one (at
    -- most one of the two is set). Both empty = text-to-video.
    image_url  text        NOT NULL DEFAULT '',
    image_id   uuid            REFERENCES images (id) ON DELETE SET NULL,
    -- The picture the clip should END on (fal's end_image_url): the clip flows
    -- from the starting picture to this one, so it only works when the two
    -- differ in the subject's pose and nothing else (same camera, same room).
    -- A stored picture only, and only with a starting picture.
    end_image_id uuid          REFERENCES images (id) ON DELETE SET NULL,
    -- The owner looked at the storyboard and OKed this plan, so it may be made
    -- in bulk. Changing the prompt, length or picture clears it (UpdateShot).
    approved   boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (video_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- One run of a prompt template through the AI provider. An immutable record:
-- the exact prompt that was sent, what came back, and which provider/model
-- produced it. Kept even if the source template is later deleted (the link is
-- nulled); removed with its profile.
CREATE TABLE IF NOT EXISTS generations (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id         uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    prompt_template_id uuid            REFERENCES prompt_templates (id) ON DELETE SET NULL,
    input_prompt       text        NOT NULL,
    -- Public image an image-to-video clip starts from ('' = text-to-video).
    image_url          text        NOT NULL DEFAULT '',
    -- ...or an uploaded one (at most one of image_url / image_id is set).
    image_id           uuid            REFERENCES images (id) ON DELETE SET NULL,
    -- The picture the clip was asked to end on, if any (see video_shots).
    end_image_id       uuid            REFERENCES images (id) ON DELETE SET NULL,
    -- Clip length asked for (3-15 s; fal's range).
    seconds            smallint    NOT NULL DEFAULT 5 CHECK (seconds BETWEEN 3 AND 15),
    -- The shot this is an attempt at, if any. A deleted shot leaves its clips
    -- behind as ordinary clips of the profile (they were paid for).
    shot_id            uuid            REFERENCES video_shots (id) ON DELETE SET NULL,
    output             text        NOT NULL DEFAULT '',
    -- 'text' -> `output` is the generated text; 'video' -> `output` is a URL.
    output_kind        text        NOT NULL DEFAULT 'text',
    status             text        NOT NULL,
    error              text        NOT NULL DEFAULT '',
    provider           text        NOT NULL,
    model              text        NOT NULL DEFAULT '',
    -- fal's id for the request that made this clip (set when the job is sent;
    -- '' for text-only runs and clips made before this column existed). It is
    -- the key for looking the real bill up in fal's billing events.
    fal_request_id     text        NOT NULL DEFAULT '',
    -- What fal actually charged, in USD, once its billing events show the
    -- request (NULL = not known yet / not a fal clip).
    cost_usd           double precision,
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS generations_profile_id_idx ON generations (profile_id);

-- The take the owner picked as THE clip of a shot (any attempt, not just the
-- newest). NULL means "the newest successful attempt". Added here, after
-- generations, because generations point back at video_shots (shot_id): the two
-- tables reference each other. A deleted attempt just clears the choice.
ALTER TABLE video_shots
    ADD COLUMN IF NOT EXISTS chosen_generation_id uuid REFERENCES generations (id) ON DELETE SET NULL;

-- Global key/value settings — provider credentials and the like. One row per
-- key, not duplicated per profile. The API whitelists which keys it accepts.
CREATE TABLE IF NOT EXISTS settings (
    key        text        PRIMARY KEY,
    value      text        NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- A YouTube connection for one channel (Phase 2 item 3). One row per channel
-- (channel_id is the PK), created by the OAuth callback and refreshed as the
-- access token expires. Tokens are stored as-is, same posture as the API key
-- in `settings`. Removed with its channel.
CREATE TABLE IF NOT EXISTS youtube_accounts (
    channel_id         uuid        PRIMARY KEY REFERENCES channels (id) ON DELETE CASCADE,
    -- The connected YouTube channel, for display ("Connected as ...").
    youtube_channel_id text        NOT NULL DEFAULT '',
    youtube_title      text        NOT NULL DEFAULT '',
    access_token       text        NOT NULL,
    -- Empty only if Google withheld it (it is returned on first consent with
    -- access_type=offline; we force prompt=consent to always get one).
    refresh_token      text        NOT NULL DEFAULT '',
    token_expiry       timestamptz NOT NULL,
    scope              text        NOT NULL DEFAULT '',
    connected_at       timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- One successful publish of a video generation to a connected YouTube
-- channel. History, not state — a generation can be published more than once
-- (e.g. to different channels). Removed with its generation or channel.
CREATE TABLE IF NOT EXISTS youtube_uploads (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id    uuid        NOT NULL REFERENCES generations (id) ON DELETE CASCADE,
    channel_id       uuid        NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    youtube_video_id text        NOT NULL,
    privacy          text        NOT NULL,
    title            text        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS youtube_uploads_generation_id_idx ON youtube_uploads (generation_id);

-- Shopee products for affiliate baskets (Phase 3). A CENTRAL catalog: a
-- product belongs to no profile or channel, so one product can ride on clips
-- from any profile and its link/price is edited in one place. Entered by hand
-- for now; shopee_shop_id/item_id are parsed from shopee_url (internal/shopee)
-- so the same product is not added twice and an API client can use them later.
CREATE TABLE IF NOT EXISTS products (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           text        NOT NULL,
    shopee_url     text        NOT NULL,
    shopee_shop_id bigint,
    shopee_item_id bigint,
    -- The owner's own tracked affiliate link, pasted from Shopee. Empty until
    -- they have one.
    affiliate_url  text        NOT NULL DEFAULT '',
    -- Whole baht, informational only. 0 = not set.
    price_thb      integer     NOT NULL DEFAULT 0 CHECK (price_thb >= 0),
    note           text        NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS products_shopee_ids_idx
    ON products (shopee_shop_id, shopee_item_id)
    WHERE shopee_shop_id IS NOT NULL AND shopee_item_id IS NOT NULL;

-- Pictures of a product, as it really looks: several angles and states (cap
-- closed, cap open, in a hand). They go to the image model as reference pictures
-- when a shot's picture is made, so the product on screen is the real one. A
-- label says what state the picture shows ("cap open"): an edit model copies the
-- state of its reference, so a shot that needs an open cap needs a picture with
-- one. At most 4 (position 1-4). Removed with the product.
CREATE TABLE IF NOT EXISTS product_images (
    product_id uuid     NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    image_id   uuid     NOT NULL REFERENCES images (id) ON DELETE CASCADE,
    position   smallint NOT NULL CHECK (position BETWEEN 1 AND 4),
    label      text     NOT NULL DEFAULT '',
    PRIMARY KEY (product_id, position),
    UNIQUE (product_id, image_id)
);

-- Products pinned to a clip (a generation) — at most two. `position` (1 or 2)
-- is the pin order and, with the primary key, is what enforces the cap in the
-- database: a third product has no free position. A product that is pinned
-- somewhere can't be deleted (RESTRICT); a deleted clip just drops its pins.
CREATE TABLE IF NOT EXISTS generation_products (
    generation_id uuid     NOT NULL REFERENCES generations (id) ON DELETE CASCADE,
    product_id    uuid     NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    position      smallint NOT NULL CHECK (position BETWEEN 1 AND 2),
    PRIMARY KEY (generation_id, position),
    UNIQUE (generation_id, product_id)
);

CREATE INDEX IF NOT EXISTS generation_products_product_id_idx ON generation_products (product_id);

-- Login sessions for the single admin user. We store only a SHA-256 hash of
-- the session token, so a leak of this table does not expose live sessions.
CREATE TABLE IF NOT EXISTS sessions (
    token_hash bytea       PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
