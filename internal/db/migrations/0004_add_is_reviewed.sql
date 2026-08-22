-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN is_reviewed BOOLEAN NOT NULL DEFAULT 0;
UPDATE transactions SET is_reviewed = 1 WHERE category_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite has limited ALTER TABLE support, so we don't drop columns easily.
-- For a full downgrade, you'd have to recreate the table.
-- +goose StatementEnd
