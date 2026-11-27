-- What the daily spending limit counts (internal/spendcap). Everything fal was
-- asked to make since `since`, with what the profile's model costs, so a clip
-- whose bill has not arrived yet can be counted at its estimate instead of $0.

-- name: SpendClipsSince :many
SELECT g.cost_usd, g.seconds, g.status, p.video_model, p.video_audio
FROM generations g
JOIN profiles p ON p.id = g.profile_id
WHERE g.provider = 'fal' AND g.created_at >= sqlc.arg(since)::timestamptz;

-- Picture requests have no stored bill; they count at the model's list price.
-- A failed request is left out (nothing was made).
-- name: SpendKeyframesSince :many
SELECT model, count
FROM keyframes
WHERE status <> 'failed' AND created_at >= sqlc.arg(since)::timestamptz;
