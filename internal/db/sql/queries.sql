-- name: GetPageToken :one
SELECT page_token
FROM state
WHERE id = 1;

-- name: GetAuthToken :one
SELECT auth_token
FROM state
WHERE id = 1;

-- name: IsInitialized :one
SELECT is_initialized
FROM state
WHERE id = 1;

-- name: UpdatePageToken :exec
UPDATE state
SET page_token = ?
WHERE id = 1;

-- name: UpdateAuthToken :exec
UPDATE state
SET auth_token = ?
WHERE id = 1;

-- name: SetInitialized :exec
UPDATE state
SET is_initialized = true;


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
INSERT INTO files (path, drive_id, content_hash, last_modified)
VALUES (?, ?, ?, ?)
ON CONFLICT (path)
    DO UPDATE SET content_hash  = EXCLUDED.content_hash,
                  last_modified = EXCLUDED.last_modified;

-- name: GetFile :one
SELECT path, drive_id, content_hash, last_modified
FROM files
WHERE path = ?;

-- name: GetAllFiles :many
SELECT path, drive_id, content_hash, last_modified
FROM files
ORDER BY path;

-- name: DeleteFile :exec
DELETE
FROM files
WHERE path = ?;
