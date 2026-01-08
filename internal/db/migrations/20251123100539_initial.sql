-- +goose Up
-- +goose StatementBegin
CREATE TABLE state (
    lock char(1) NOT NULL DEFAULT('X') PRIMARY KEY CHECK (lock IN ('X')),
    current_page_token text NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE state;
-- +goose StatementEnd
