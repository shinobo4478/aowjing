-- name: ListPromptTemplatesByProfile :many
SELECT * FROM prompt_templates
WHERE profile_id = $1
ORDER BY created_at DESC;

-- name: GetPromptTemplate :one
SELECT * FROM prompt_templates
WHERE id = $1;

-- name: CreatePromptTemplate :one
INSERT INTO prompt_templates (profile_id, name, body, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePromptTemplate :one
UPDATE prompt_templates
SET name = $2,
    body = $3,
    description = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePromptTemplate :execrows
DELETE FROM prompt_templates
WHERE id = $1;
