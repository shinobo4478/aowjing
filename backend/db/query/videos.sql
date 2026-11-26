-- name: CreateVideo :one
INSERT INTO videos (profile_id, title)
VALUES ($1, $2)
RETURNING *;

-- name: GetVideo :one
SELECT * FROM videos
WHERE id = $1;

-- name: ListVideosByProfile :many
SELECT v.id, v.profile_id, v.title, v.created_at, v.updated_at,
       COUNT(s.id)::int AS shot_count,
       COALESCE(SUM(s.seconds), 0)::int AS total_seconds
FROM videos v
LEFT JOIN video_shots s ON s.video_id = v.id
WHERE v.profile_id = $1
GROUP BY v.id
ORDER BY v.created_at DESC;

-- name: UpdateVideoTitle :one
UPDATE videos
SET title = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteVideo :execrows
DELETE FROM videos
WHERE id = $1;

-- Every picture of a video's shots, collected BEFORE the video is deleted so
-- the ones nothing else uses can be removed afterwards.
-- name: ListVideoImageIDs :many
SELECT s.image_id AS image_id
FROM video_shots s
WHERE s.video_id = $1 AND s.image_id IS NOT NULL
UNION
SELECT e.end_image_id
FROM video_shots e
WHERE e.video_id = $1 AND e.end_image_id IS NOT NULL;

-- name: ListShots :many
SELECT * FROM video_shots
WHERE video_id = $1
ORDER BY position;

-- name: GetShot :one
SELECT * FROM video_shots
WHERE id = $1;

-- name: CountShots :one
SELECT COUNT(*)::int FROM video_shots
WHERE video_id = $1;

-- Adds a shot at the end.
-- name: CreateShot :one
INSERT INTO video_shots (video_id, position, prompt, seconds)
VALUES (
    $1,
    (SELECT COALESCE(MAX(position), 0) + 1 FROM video_shots WHERE video_id = $1),
    $2,
    $3
)
RETURNING *;

-- Adds a drafted shot (from the script helper) at the end, with its picture
-- prompt already filled in.
-- name: CreateShotWithPlan :one
INSERT INTO video_shots (video_id, position, prompt, seconds, picture_prompt, end_picture_prompt)
VALUES (
    $1,
    (SELECT COALESCE(MAX(position), 0) + 1 FROM video_shots WHERE video_id = $1),
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- Changing the plan clears the approval: it applied to the plan that was
-- looked at, not to whatever the shot becomes.
-- name: UpdateShot :one
UPDATE video_shots
SET approved = CASE
        WHEN prompt = $2 AND seconds = $3 AND image_url = $4
             AND image_id IS NOT DISTINCT FROM sqlc.narg('image_id')::uuid
             AND end_image_id IS NOT DISTINCT FROM sqlc.narg('end_image_id')::uuid
        THEN approved ELSE false END,
    prompt = $2,
    seconds = $3,
    image_url = $4,
    image_id = sqlc.narg('image_id')::uuid,
    end_image_id = sqlc.narg('end_image_id')::uuid,
    picture_prompt = sqlc.arg('picture_prompt')::text,
    end_picture_prompt = sqlc.arg('end_picture_prompt')::text,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetShotApproved :one
UPDATE video_shots
SET approved = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteShot :execrows
DELETE FROM video_shots
WHERE id = $1;

-- Closes the gap a deleted shot leaves: 1, 2, 4 becomes 1, 2, 3. One statement;
-- the unique constraint on position is deferred, so the temporary overlap while
-- it runs is fine.
-- name: CompactShotPositions :exec
UPDATE video_shots s
SET position = r.n
FROM (
    SELECT t.id, ROW_NUMBER() OVER (ORDER BY t.position)::smallint AS n
    FROM video_shots t
    WHERE t.video_id = $1
) r
WHERE s.id = r.id AND s.position <> r.n;

-- name: SetShotPosition :exec
UPDATE video_shots
SET position = $3
WHERE id = $1 AND video_id = $2;

-- Every attempt at one shot, newest first, for choosing the take to use.
-- name: ListShotAttempts :many
SELECT g.id AS id,
       g.status AS status,
       g.output AS output,
       g.error AS error,
       g.seconds AS seconds,
       g.created_at AS created_at
FROM generations g
WHERE g.shot_id = $1
ORDER BY g.created_at DESC;

-- Picks (or, with NULL, clears) the take of a shot. The handler has checked the
-- attempt belongs to the shot and succeeded.
-- name: SetShotChosen :one
UPDATE video_shots
SET chosen_generation_id = sqlc.narg('generation_id')::uuid,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- The take of each shot of a video: the one the owner chose if it succeeded,
-- else the newest successful attempt. `chosen` says which of the two it is.
-- name: ListShotTakes :many
SELECT DISTINCT ON (s.id)
       s.id AS shot_id,
       g.id AS id,
       g.status AS status,
       g.output AS output,
       g.error AS error,
       g.seconds AS seconds,
       g.created_at AS created_at,
       COALESCE(g.id = s.chosen_generation_id, false)::boolean AS chosen
FROM video_shots s
JOIN generations g ON g.shot_id = s.id AND g.status = 'succeeded'
WHERE s.video_id = $1
ORDER BY s.id, COALESCE(g.id = s.chosen_generation_id, false) DESC, g.created_at DESC;

-- The newest attempt at each shot of a video, and how many attempts there have
-- been, so the page can show each shot's current result without a request per
-- shot.
-- name: ListShotLatestGenerations :many
SELECT DISTINCT ON (g.shot_id)
       g.shot_id AS shot_id,
       g.id AS id,
       g.status AS status,
       g.output AS output,
       g.error AS error,
       g.seconds AS seconds,
       g.created_at AS created_at,
       (SELECT COUNT(*) FROM generations x WHERE x.shot_id = g.shot_id)::int AS attempts
FROM generations g
WHERE g.shot_id IN (SELECT id FROM video_shots WHERE video_id = $1)
ORDER BY g.shot_id, g.created_at DESC;
