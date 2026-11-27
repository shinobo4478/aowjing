-- name: ListGenerationsByProfile :many
SELECT g.*, pt.name AS template_name
FROM generations g
LEFT JOIN prompt_templates pt ON pt.id = g.prompt_template_id
WHERE g.profile_id = $1
ORDER BY g.created_at DESC;

-- name: GetGeneration :one
SELECT g.*, pt.name AS template_name
FROM generations g
LEFT JOIN prompt_templates pt ON pt.id = g.prompt_template_id
WHERE g.id = $1;

-- name: CreateGeneration :one
INSERT INTO generations (
    profile_id, prompt_template_id, input_prompt, image_url, image_id, output, status, error, provider, model,
    seconds, shot_id, end_image_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: FinishGeneration :exec
UPDATE generations
SET status = $2,
    output = $3,
    output_kind = $4,
    error = $5,
    provider = $6,
    model = $7,
    fal_request_id = $8
WHERE id = $1;

-- name: ListGenerationsToCost :many
-- Clips whose real cost we still need to ask fal for.
SELECT id, fal_request_id, created_at
FROM generations
WHERE id = ANY($1::uuid[])
  AND fal_request_id <> ''
  AND cost_usd IS NULL;

-- name: SetGenerationCost :exec
UPDATE generations SET cost_usd = $2 WHERE id = $1;

-- name: DeleteGeneration :execrows
DELETE FROM generations
WHERE id = $1;
