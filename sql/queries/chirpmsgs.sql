-- name: GetChirps :many
SELECT id, created_at, updated_at, body, user_id FROM chirpmsgs
ORDER BY created_at ASC;


-- name: GetChirpById :one  
SELECT id, created_at, updated_at, body, user_id FROM chirpmsgs
WHERE id = $1;


-- name: CreateChirp :one
INSERT INTO chirpmsgs (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;
