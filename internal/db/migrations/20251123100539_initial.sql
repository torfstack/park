-- +goose Up
-- +goose StatementBegin
CREATE TABLE files (
   id INTEGER PRIMARY KEY,
   path text NOT NULL,
   content_hash bytea NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE files;
-- +goose StatementEnd
