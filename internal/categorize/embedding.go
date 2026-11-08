package categorize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/Cakem1x/fin_man/internal/model"
)

const embeddingBatchSize = 64

// embeddingInstruction is prepended to each EmbeddingText in the instruction
// format of qwen3-embedding. It moves texts of the same merchant closer
// together and texts that only share bank boilerplate further apart.
const embeddingInstruction = "Instruct: Given a bank transaction, retrieve transactions from the same merchant or of the same spending category\nQuery: "

// EmbeddingStore caches embeddings per model, keyed by transaction ID.
type EmbeddingStore interface {
	GetEmbeddings(ctx context.Context, embedModel string) (map[string]model.Embedding, error)
	UpsertEmbeddings(ctx context.Context, embedModel string, embs map[string]model.Embedding) error
}

// Embedder computes one embedding vector per input.
type Embedder interface {
	Embed(ctx context.Context, embedModel string, inputs []string) ([][]float32, error)
}

// EmbeddingText returns the payee and memo without words that contain digits.
// Such words are dates, IBANs, card numbers and references, which would
// otherwise dominate the embedding.
func EmbeddingText(tx model.Transaction) string {
	var words []string
	for _, w := range strings.Fields(tx.Payee + " " + tx.Memo) {
		if !strings.ContainsFunc(w, unicode.IsDigit) {
			words = append(words, w)
		}
	}
	return strings.Join(words, " ")
}

// embeddingInput returns the text sent to the embedding model, or "" if
// EmbeddingText is empty.
func embeddingInput(tx model.Transaction) string {
	text := EmbeddingText(tx)
	if text == "" {
		return ""
	}
	return embeddingInstruction + text
}

func textHash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

// LoadEmbeddings returns the cached vectors of txs whose embedding input is
// unchanged since embedding, keyed by transaction ID.
func LoadEmbeddings(ctx context.Context, store EmbeddingStore, embedModel string, txs []model.Transaction) (map[string][]float32, error) {
	cached, err := store.GetEmbeddings(ctx, embedModel)
	if err != nil {
		return nil, err
	}
	vecs := make(map[string][]float32)
	for _, tx := range txs {
		if e, ok := cached[tx.ID]; ok && e.TextHash == textHash(embeddingInput(tx)) {
			vecs[tx.ID] = e.Vector
		}
	}
	return vecs, nil
}

// EnsureEmbeddings embeds and stores all txs without an up-to-date cached
// vector. Transactions with an empty EmbeddingText are skipped. progress, if
// not nil, is called after each stored batch. It returns the vectors of all
// txs, keyed by transaction ID.
func EnsureEmbeddings(ctx context.Context, store EmbeddingStore, embedder Embedder, embedModel string, txs []model.Transaction, progress func(done, total int)) (map[string][]float32, error) {
	vecs, err := LoadEmbeddings(ctx, store, embedModel, txs)
	if err != nil {
		return nil, err
	}
	var missingIDs, missingInputs []string
	for _, tx := range txs {
		if _, ok := vecs[tx.ID]; ok {
			continue
		}
		if input := embeddingInput(tx); input != "" {
			missingIDs = append(missingIDs, tx.ID)
			missingInputs = append(missingInputs, input)
		}
	}

	for start := 0; start < len(missingIDs); start += embeddingBatchSize {
		end := min(start+embeddingBatchSize, len(missingIDs))
		batch, err := embedder.Embed(ctx, embedModel, missingInputs[start:end])
		if err != nil {
			return nil, err
		}
		embs := make(map[string]model.Embedding, len(batch))
		for i, vec := range batch {
			id := missingIDs[start+i]
			embs[id] = model.Embedding{TextHash: textHash(missingInputs[start+i]), Vector: vec}
			vecs[id] = vec
		}
		if err := store.UpsertEmbeddings(ctx, embedModel, embs); err != nil {
			return nil, err
		}
		if progress != nil {
			progress(end, len(missingIDs))
		}
	}
	return vecs, nil
}
