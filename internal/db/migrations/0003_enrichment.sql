-- +goose Up
-- +goose StatementBegin
CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE transaction_tags (
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
);

ALTER TABLE transactions ADD COLUMN category_id TEXT REFERENCES categories(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite has limited ALTER TABLE support, so we usually don't drop columns easily,
-- but we can drop the tables.
DROP TABLE transaction_tags;
DROP TABLE tags;
DROP TABLE categories;
-- +goose StatementEnd
