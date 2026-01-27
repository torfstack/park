-- +goose Up
-- +goose StatementBegin
CREATE TABLE state
(
    id                 int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    current_page_token text NOT NULL
);

CREATE TABLE config
(
    id            int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    root_dir      text NOT NULL,
    sync_interval int  NOT NULL
);

CREATE TABLE files
(
    path          text PRIMARY KEY,
    drive_id      text NOT NULL,
    content_hash  blob NOT NULL,
    last_modified int  NOT NULL
);

INSERT INTO state (id, current_page_token)
VALUES (1, '');

INSERT INTO config (id, root_dir, sync_interval)
VALUES (1, '', 3600);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE state;
DROP TABLE config;
DROP TABLE files;
-- +goose StatementEnd
