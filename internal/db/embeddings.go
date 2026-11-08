package db

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/Cakem1x/fin_man/internal/model"
)

// GetEmbeddings returns all cached embeddings of the model, keyed by transaction ID.
func (db *DB) GetEmbeddings(ctx context.Context, embedModel string) (map[string]model.Embedding, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT transaction_id, text_hash, vector FROM transaction_embeddings WHERE model = ?`, embedModel)
	if err != nil {
		return nil, fmt.Errorf("query embeddings failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	embs := make(map[string]model.Embedding)
	for rows.Next() {
		var txID, hash string
		var blob []byte
		if err := rows.Scan(&txID, &hash, &blob); err != nil {
			return nil, fmt.Errorf("scan embedding failed: %w", err)
		}
		embs[txID] = model.Embedding{TextHash: hash, Vector: decodeVector(blob)}
	}
	return embs, rows.Err()
}

// UpsertEmbeddings stores embeddings of the model, keyed by transaction ID.
func (db *DB) UpsertEmbeddings(ctx context.Context, embedModel string, embs map[string]model.Embedding) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO transaction_embeddings (transaction_id, model, text_hash, vector)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(transaction_id, model) DO UPDATE SET text_hash = excluded.text_hash, vector = excluded.vector
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for txID, e := range embs {
		if _, err := stmt.ExecContext(ctx, txID, embedModel, e.TextHash, encodeVector(e.Vector)); err != nil {
			return fmt.Errorf("failed to upsert embedding for %s: %w", txID, err)
		}
	}
	return tx.Commit()
}

// encodeVector serializes the vector as little-endian float32 values.
func encodeVector(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return b
}

func decodeVector(b []byte) []float32 {
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}
