package categorize

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestEmbeddingText(t *testing.T) {
	tests := []struct {
		name, payee, memo, want string
	}{
		{"payee and memo", "Stadtwerke München", "Abschlag Strom", "Stadtwerke München Abschlag Strom"},
		{"drops tokens with digits", "REWE Markt GmbH", "2024-01-05T12:34 Debitk.12 VISA DE89370400440532013000", "REWE Markt GmbH VISA"},
		{"collapses whitespace", "  Netflix ", "\tB.V.  ", "Netflix B.V."},
		{"empty memo", "Spotify AB", "", "Spotify AB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EmbeddingText(model.Transaction{Payee: tt.payee, Memo: tt.memo})
			if got != tt.want {
				t.Errorf("EmbeddingText = %q, want %q", got, tt.want)
			}
		})
	}
}

type fakeStore map[string]model.Embedding

func (s fakeStore) GetEmbeddings(_ context.Context, embedModel string) (map[string]model.Embedding, error) {
	out := make(map[string]model.Embedding)
	for k, v := range s {
		if strings.HasPrefix(k, embedModel+"/") {
			out[strings.TrimPrefix(k, embedModel+"/")] = v
		}
	}
	return out, nil
}

func (s fakeStore) UpsertEmbeddings(_ context.Context, embedModel string, embs map[string]model.Embedding) error {
	for id, e := range embs {
		s[embedModel+"/"+id] = e
	}
	return nil
}

type fakeEmbedder struct{ calls [][]string }

func (f *fakeEmbedder) Embed(_ context.Context, _ string, inputs []string) ([][]float32, error) {
	f.calls = append(f.calls, inputs)
	vecs := make([][]float32, len(inputs))
	for i, in := range inputs {
		vecs[i] = []float32{float32(len(in))}
	}
	return vecs, nil
}

func TestEnsureEmbeddings(t *testing.T) {
	ctx := context.Background()
	store := fakeStore{}
	var txs []model.Transaction
	for i := range 70 {
		txs = append(txs, model.Transaction{ID: fmt.Sprint("tx", i), Payee: strings.Repeat("a", i+1)})
	}
	txs = append(txs, model.Transaction{ID: "digits-only", Payee: "12345"})

	emb := &fakeEmbedder{}
	var progress []int
	vecs, err := EnsureEmbeddings(ctx, store, emb, "m", txs, func(done, total int) {
		if total != 70 {
			t.Errorf("total = %d, want 70", total)
		}
		progress = append(progress, done)
	})
	if err != nil {
		t.Fatalf("EnsureEmbeddings: %v", err)
	}
	if len(emb.calls) != 2 || len(emb.calls[0]) != embeddingBatchSize || len(emb.calls[1]) != 70-embeddingBatchSize {
		t.Errorf("unexpected batches: %d calls", len(emb.calls))
	}
	if !reflect.DeepEqual(progress, []int{embeddingBatchSize, 70}) {
		t.Errorf("progress = %v", progress)
	}
	if len(vecs) != 70 || vecs["tx2"][0] != float32(len(embeddingInstruction+"aaa")) {
		t.Errorf("unexpected vectors: %d, tx2=%v", len(vecs), vecs["tx2"])
	}

	txs[0].Memo = "edited"
	emb.calls = nil
	vecs, err = EnsureEmbeddings(ctx, store, emb, "m", txs, nil)
	if err != nil {
		t.Fatalf("EnsureEmbeddings second run: %v", err)
	}
	if !reflect.DeepEqual(emb.calls, [][]string{{embeddingInstruction + "a edited"}}) {
		t.Errorf("second run embedded %v, want only the edited transaction", emb.calls)
	}
	if len(vecs) != 70 || vecs["tx0"][0] != float32(len(embeddingInstruction+"a edited")) {
		t.Errorf("unexpected vectors after edit: %d, tx0=%v", len(vecs), vecs["tx0"])
	}
}

func TestLoadEmbeddingsIgnoresStaleText(t *testing.T) {
	ctx := context.Background()
	store := fakeStore{}
	txs := []model.Transaction{{ID: "a", Payee: "REWE"}, {ID: "b", Payee: "EDEKA"}}
	if _, err := EnsureEmbeddings(ctx, store, &fakeEmbedder{}, "m", txs, nil); err != nil {
		t.Fatal(err)
	}
	txs[1].Memo = "edited"

	vecs, err := LoadEmbeddings(ctx, store, "m", txs)
	if err != nil {
		t.Fatalf("LoadEmbeddings: %v", err)
	}
	if _, ok := vecs["b"]; ok || len(vecs) != 1 {
		t.Errorf("LoadEmbeddings = %v, want only a", vecs)
	}
	if other, _ := LoadEmbeddings(ctx, store, "other", txs); len(other) != 0 {
		t.Errorf("vectors of another model returned: %v", other)
	}
}
