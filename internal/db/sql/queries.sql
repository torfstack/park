-- name: GetState :one
SELECT id, current_page_token
FROM state
WHERE id = 1;

-- name: UpsertPageToken :exec
INSERT INTO state (id, current_page_token)
VALUES (?, ?)
ON CONFLICT (id) DO UPDATE SET current_page_token = EXCLUDED.current_page_token;

-- name: GetConfig :one
SELECT id, root_dir, sync_interval
FROM config
WHERE id = 1;

-- name: UpsertConfig :exec
INSERT INTO config (root_dir, sync_interval)
VALUES (?, ?)
ON CONFLICT (id) DO UPDATE SET root_dir      = EXCLUDED.root_dir,
                               sync_interval = EXCLUDED.sync_interval;

-- name: UpsertFile :exec
INSERT INTO files (path, content_hash, last_modified)
VALUES (?, ?, ?)
ON CONFLICT (path)
    DO UPDATE SET content_hash  = EXCLUDED.content_hash,
                  last_modified = EXCLUDED.last_modified;

-- name: GetFile :one
SELECT path, content_hash, last_modified
FROM files
WHERE path = ?;

-- name: GetAllFiles :many
SELECT path, content_hash, last_modified
FROM files
ORDER BY path;

-- name: DeleteFile :exec
DELETE
FROM files
WHERE path = ?;
