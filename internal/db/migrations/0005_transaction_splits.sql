-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN parent_id TEXT REFERENCES transactions(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite has limited ALTER TABLE support, so we don't drop columns easily.
-- +goose StatementEnd
