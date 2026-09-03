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
    profile_id, prompt_template_id, input_prompt, output, status, error, provider, model
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: FinishGeneration :exec
UPDATE generations
SET status = $2,
    output = $3,
    error = $4,
    provider = $5,
    model = $6
WHERE id = $1;

-- name: DeleteGeneration :execrows
DELETE FROM generations
WHERE id = $1;
