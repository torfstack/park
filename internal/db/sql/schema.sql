CREATE TABLE state
(
    id             int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    page_token     text NOT NULL,
    auth_token     text NOT NULL,
    is_initialized bool NOT NULL
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
