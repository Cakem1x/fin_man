-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN archive_file_path TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP COLUMN archive_file_path;
-- +goose StatementEnd
