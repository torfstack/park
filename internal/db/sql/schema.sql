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
    content_hash  text NOT NULL,
    last_modified int  NOT NULL
);
