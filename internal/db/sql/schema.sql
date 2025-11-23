CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    path text NOT NULL,
    content_hash bytea NOT NULL
)