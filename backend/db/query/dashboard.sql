-- Numbers for the home dashboard. Every query that groups by day or month takes
-- the viewer's time zone (an IANA name such as Asia/Bangkok) so "today" and
-- "this month" mean the same thing on screen as on the clock.

-- name: DashToday :one
SELECT (now() AT TIME ZONE sqlc.arg(tz)::text)::date AS today;

-- What fal charged: today, this month and all time. Clips whose bill fal has not shown
-- yet (or that predate request-id tracking) are counted apart, never as $0.
-- name: DashSpend :one
SELECT
    COALESCE(SUM(cost_usd) FILTER (WHERE created_at >= (date_trunc('day', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)), 0)::float8 AS today_cost,
    (COUNT(cost_usd) FILTER (WHERE created_at >= (date_trunc('day', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)))::bigint AS today_billed,
    (COUNT(*) FILTER (WHERE cost_usd IS NULL AND created_at >= (date_trunc('day', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)))::bigint AS today_unbilled,
    COALESCE(SUM(cost_usd) FILTER (WHERE created_at >= (date_trunc('month', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)), 0)::float8 AS month_cost,
    (COUNT(cost_usd) FILTER (WHERE created_at >= (date_trunc('month', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)))::bigint AS month_billed,
    (COUNT(*) FILTER (WHERE cost_usd IS NULL AND created_at >= (date_trunc('month', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)))::bigint AS month_unbilled,
    COALESCE(SUM(cost_usd), 0)::float8 AS total_cost,
    COUNT(cost_usd)::bigint AS total_billed,
    (COUNT(*) FILTER (WHERE cost_usd IS NULL))::bigint AS total_unbilled
FROM generations
WHERE provider = 'fal' AND status <> 'pending';

-- name: DashSpendByDay :many
SELECT (created_at AT TIME ZONE sqlc.arg(tz)::text)::date AS day,
       SUM(cost_usd)::float8 AS cost,
       COUNT(*)::bigint AS clips
FROM generations
WHERE provider = 'fal' AND cost_usd IS NOT NULL
  AND created_at >= now() - interval '40 days'
GROUP BY 1
ORDER BY 1;

-- name: DashSpendByProfile :many
SELECT p.id, p.name,
       COALESCE(SUM(g.cost_usd), 0)::float8 AS cost,
       COUNT(g.cost_usd)::bigint AS clips
FROM profiles p
JOIN generations g ON g.profile_id = p.id
WHERE g.provider = 'fal' AND g.cost_usd IS NOT NULL
  AND g.created_at >= (date_trunc('month', now() AT TIME ZONE sqlc.arg(tz)::text) AT TIME ZONE sqlc.arg(tz)::text)
GROUP BY p.id, p.name
ORDER BY cost DESC
LIMIT 8;

-- name: DashPending :many
SELECT g.id, g.profile_id, p.name AS profile_name, g.seconds, g.shot_id, g.created_at
FROM generations g
JOIN profiles p ON p.id = g.profile_id
WHERE g.status = 'pending'
ORDER BY g.created_at
LIMIT 10;

-- name: DashRecentFailed :many
SELECT g.id, g.profile_id, p.name AS profile_name, g.error, g.cost_usd, g.created_at
FROM generations g
JOIN profiles p ON p.id = g.profile_id
WHERE g.status = 'failed'
ORDER BY g.created_at DESC
LIMIT 5;

-- name: DashPipeline :one
SELECT
    (SELECT COUNT(*) FROM videos)::bigint AS videos,
    (SELECT COUNT(*) FROM video_shots)::bigint AS shots,
    (SELECT COUNT(*) FROM video_shots WHERE approved)::bigint AS shots_approved,
    -- OKed on the storyboard but never made: the next spend waiting to happen.
    (SELECT COUNT(*) FROM video_shots s WHERE s.approved
        AND NOT EXISTS (SELECT 1 FROM generations g WHERE g.shot_id = s.id))::bigint AS approved_not_made,
    (SELECT COUNT(*) FROM video_shots s
        WHERE EXISTS (SELECT 1 FROM generations g WHERE g.shot_id = s.id AND g.status = 'succeeded'))::bigint AS shots_made,
    -- Finished videos nobody has published (retakes count too).
    (SELECT COUNT(*) FROM generations g
        WHERE g.status = 'succeeded' AND g.output_kind = 'video'
          AND NOT EXISTS (SELECT 1 FROM youtube_uploads u WHERE u.generation_id = g.id))::bigint AS unpublished_clips;

-- name: DashPublishedCounts :one
SELECT
    COUNT(*) FILTER (WHERE created_at >= now() - interval '7 days')::bigint AS last_7_days,
    COUNT(*) FILTER (WHERE created_at >= now() - interval '30 days')::bigint AS last_30_days,
    COUNT(*)::bigint AS total
FROM youtube_uploads;

-- name: DashRecentUploads :many
SELECT u.id, u.title, u.privacy, u.youtube_video_id, u.created_at,
       c.name AS channel_name, p.id AS profile_id, p.name AS profile_name
FROM youtube_uploads u
JOIN channels c ON c.id = u.channel_id
JOIN profiles p ON p.id = c.profile_id
ORDER BY u.created_at DESC
LIMIT 5;

-- name: DashProducts :one
SELECT
    COUNT(*)::bigint AS total,
    (COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM generation_products gp WHERE gp.product_id = products.id)))::bigint AS pinned
FROM products;

-- name: DashUnpinnedProducts :many
SELECT id, name, created_at
FROM products
WHERE NOT EXISTS (SELECT 1 FROM generation_products gp WHERE gp.product_id = products.id)
ORDER BY created_at DESC
LIMIT 5;
