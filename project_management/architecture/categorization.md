# Categorization

## Pipeline

1. similarity suggestions
2. manual review

There are no deterministic categorization rules.
The user confirms every categorization.
The estimator learns from each confirmed transaction.

## Similarity Suggestions

The local estimator compares the payee and memo of a transaction with each
previously categorized transaction. The similarity is the larger of the
token-set Jaccard similarity and the embedding similarity (see
[Embedding Similarity](#embedding-similarity)). The estimator raises the
similarity to the
third power (`weight = similarity^3`). This lets close matches contribute much
more than weak overlaps while still combining evidence from every transaction.

For each category, the estimator sums the weights of transactions assigned to
that category and divides by the total category weight plus a smoothing value
of `0.1`. Category scores therefore account for competing categories and may
sum to less than one when the match evidence is weak.

Tags are scored independently. For a tag, matching transactions contribute
positive weight and transactions without that tag contribute negative weight.
The score is `positive / (positive + negative + 0.1)`. This lets both repeated
positive examples and nearby counterexamples affect the result.

## Embedding Similarity

Embedding similarity finds related transactions without shared words, e.g.
`REWE SAGT DANKE` and `REWE Markt GmbH`. It is optional (see
[Configuration](#configuration)). If it is disabled, Ollama is unreachable, or
a transaction has no embedding, the estimator uses Jaccard similarity only.

The embedding text is the payee and memo without words that contain digits
(dates, IBANs, card numbers, references). The model input is the embedding text
with a qwen3-embedding instruction prefix.

The embedding similarity is the cosine similarity, rescaled from `[0.7, 1]` to
`[0, 1]`; lower values become `0`. Texts of unrelated merchants reach cosine
values of about 0.4 to 0.65. Without the rescaling, all past transactions would
contribute weight and dilute close matches.

The `transaction_embeddings` table caches one vector per transaction and model,
with a hash of the model input. A vector whose hash does not match the current
model input, e.g. after a memo edit, is ignored and recomputed.

Embeddings are computed:

- by `fin classify`, for all transactions without an up-to-date embedding
- at the end of `fin import-csv`, if new transactions were inserted

`fin tui` loads the cached vectors at startup. It does not compute embeddings.

## Configuration

The optional `[llm]` section in `fin.toml` configures the Ollama integration.
Missing keys use these defaults:

```toml
[llm]
enabled = false
endpoint = "http://localhost:11434"
embed_model = "qwen3-embedding:0.6b"
chat_model = "qwen3:1.7b"
fallback_threshold = 0.8
```

`chat_model` and `fallback_threshold` are reserved for the planned generative
fallback.

Ollama runs as a separate service. Models are pulled into `~/.ollama/models`.

- Development: `just ollama-serve [models...]` (`nix run .#ollama-serve`)
  starts `ollama serve` if no server answers on `$OLLAMA_HOST` (default
  `127.0.0.1:11434`), then pulls the models. Without arguments it pulls
  `embed_model` and `chat_model` defaults.
- NixOS: `services.ollama = { enable = true; loadModels = [ "qwen3-embedding:0.6b" "qwen3:1.7b" ]; };`

## Review Workflow

The user reviews proposed categories in the TUI and can:

- accept a suggestion
- pick or create a category
- pick or create tags
- edit the note

The review lists are ordered by score, with the original option order used to
break ties. Scores are shown as percentages. A category or tag is preselected
when its score is greater than 80%; lower scoring options remain available for
manual selection. Existing tags on a transaction remain selected.
