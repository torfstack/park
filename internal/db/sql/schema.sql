CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    path text NOT NULL,
    parent_dir_id INTEGER REFERENCES dirs(id),
    drive_id text NOT NULL,
    content_hash bytea
);

CREATE TABLE dirs (
    id INTEGER PRIMARY KEY,
    parent_dir_id INTEGER REFERENCES dirs(id),
    drive_id text NOT NULL,
    path text NOT NULL
);

CREATE TABLE outbox (
    id INTEGER PRIMARY KEY,
    file_id INTEGER REFERENCES files(id),
    status text NOT NULL
)