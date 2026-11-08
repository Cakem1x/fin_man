-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_embeddings (
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    text_hash TEXT NOT NULL,
    vector BLOB NOT NULL,
    PRIMARY KEY (transaction_id, model)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE transaction_embeddings;
-- +goose StatementEnd
