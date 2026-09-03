-- name: CreateSession :exec
INSERT INTO sessions (token_hash, expires_at)
VALUES ($1, $2);

-- name: GetSession :one
SELECT token_hash, created_at, expires_at FROM sessions
WHERE token_hash = $1 AND expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at <= now();
