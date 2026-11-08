# Backlog: Local LLM–assisted transaction classification (Ollama)

## Context
Classification currently uses a distance-weighted k-NN over token-set Jaccard similarity of payee+memo (`internal/categorize/estimator.go`). It works well for repeat payees but fails when no tokens overlap (e.g. "REWE Markt" vs "EDEKA Center", new payees, abbreviations). Goal: use small local models via Ollama — data stays on the machine — to improve suggestions, without changing the review UX contract (ranked lists, 80% preselect, user confirms).

Decisions (from user):
- **Phased**: Phase 1 = embedding similarity inside k-NN. Phase 2 = generative LLM fallback when k-NN confidence is low.
- **Timing**: both batch (`fin classify`, also after import) and on-demand async in the review TUI.
- **Labels**: model may only choose existing categories/tags.
- Feature is opt-in via `fin.toml`; if disabled or Ollama unreachable, behavior is exactly as today.

Follow AGENTS.md: TDD (failing test first), `just lint/build/test`, `nix flake check` at the end. Ask whether to run in a herdr worktree (large task).

## Related tasks
Prefer doing these first; adapt the TUI/cmd parts below to their result:
- `app_service_layer.md`: estimator and embedding/LLM calls belong in `internal/app.Service`, not `OverviewModel`.
- `review_draft_model.md`, `tui_edit_view.md`: async suggestion updates must target the new draft/edit view.
- `category_tag_ids.md`: online learning with real IDs.
- `cmd_workspace_db_helper.md`: use the shared helper in `fin classify`.

## Docs
- At the end, update `project_management/architecture/categorization.md` (single source of truth for the algorithm): embedding similarity, LLM fallback, config.

## Shared groundwork
1. **Ollama client** – new package `internal/ollama` (stdlib `net/http` only, no new deps):
   - `Client{BaseURL, HTTPClient}`; `Embed(ctx, model string, inputs []string) ([][]float32, error)` → `POST /api/embed`.
   - `ChatJSON(ctx, model string, msgs []Message, schema any) ([]byte, error)` → `POST /api/chat` with `stream:false`, `format:<json schema>`, `options.temperature:0`, `think:false`.
   - Tests with `httptest.Server` (request shape, error/timeout handling).
2. **Config** – extend `workspace.Config` (`internal/workspace/workspace.go`) with optional section; export `LoadConfig` (currently private `loadConfig`):
   ```toml
   [llm]
   enabled = true
   endpoint = "http://localhost:11434"
   embed_model = "qwen3-embedding:0.6b"   # multilingual (DKB data is German)
   chat_model = "qwen3:1.7b"              # CPU-only laptop; qwen3:4b optional for batch quality
   fallback_threshold = 0.8               # top k-NN category score below this triggers Phase 2
   ```
   Defaults applied when keys missing; `enabled=false` by default. Existing fin.toml files must still parse.
3. **Embedding text** – `categorize.EmbeddingText(tx)` (payee + memo, strip digit runs/IBAN/reference noise similar to `tokenize`). Unit-tested, since boilerplate dominates embeddings otherwise.

## Phase 1 – Embedding similarity in k-NN
1. **Migration** `internal/db/migrations/0006_embeddings.sql`:
   `transaction_embeddings(transaction_id TEXT REFERENCES transactions(id), model TEXT, text_hash TEXT, vector BLOB, PRIMARY KEY(transaction_id, model))`. `text_hash` invalidates the cache when the memo is edited.
2. **DB** (`internal/db/db.go`): `GetEmbeddings(ctx, model) (map[string][]float32, error)` (only rows whose hash matches current text), `UpsertEmbeddings(ctx, model, map[txID]{hash, vec})`, `GetTransactionsMissingEmbedding(ctx, model)`. Vectors as little-endian float32 BLOB.
3. **Estimator** (`internal/categorize/estimator.go`):
   - Store tx ID in `trainedData`; add optional vector store: `NewEstimator(corpus, WithEmbeddings(vecs))`, `SetEmbedding(txID, vec)`; `Learn` unchanged signature.
   - Similarity: `S = max(jaccard, rescaledCosine)` where `rescaledCosine = clamp((cos - floor)/(1 - floor), 0, 1)`, `floor` constant (~0.5, tune on real data). Rescaling is required because unrelated texts still have cosine 0.3–0.6. If either vector is missing, fall back to Jaccard only. Voting/`S^3`/smoothing unchanged.
   - TDD: tests for "no token overlap but close vectors → category wins", "missing vector → identical to current results" (regression guard).
4. **Embedding service** – `internal/categorize` (or new `internal/assist`) function `EnsureEmbeddings(ctx, db, client, model, txs)`: embeds missing ones in batches of ~64, upserts.
5. **Batch**: new `cmd/fin/cmd/classify.go` (`fin classify`): open workspace/db like `tui.go`, embed all missing transactions with progress output. Call the same function best-effort at the end of `import_csv.go` when enabled (warn, don't fail, if Ollama is down).
6. **TUI** (`internal/tui/overview.go`): build estimator with cached vectors at startup. In `openReviewForm`, if the tx has no vector, return an async `tea.Cmd` that embeds+stores it and emits `embeddingReadyMsg{txID, vec}`; on receipt (and if still reviewing that tx) call `estimator.SetEmbedding`, re-estimate, and `reviewModel.UpdateSuggestion(est)`.
   - New `ReviewModel.UpdateSuggestion` (`internal/tui/review.go`): re-sort lists and update score descriptions, preserving cursor item and any selection the user already changed; apply 80% preselect only if the user has not touched that list. Extract the current sort/preselect logic from `NewReviewModel` into a shared function so both paths use it. Tests in `review_test.go`.
   - Online learning in `saveReview` also passes the vector (already cached).

## Phase 2 – Generative fallback for low confidence
1. **Classifier** `categorize.LLMClassifier` (new file): input = target tx, category list, tag list, top-N (≈8) nearest reviewed tx from the estimator (add `Estimator.Neighbors(tx, n)`) as few-shot examples (payee, memo, amount, category, tags).
   - JSON schema: `{"category": {"enum": [<existing>..., ""]}, "tags": {"type":"array","items":{"enum":[<existing tags>]}}}` → guarantees existing labels only. Validate response anyway (drop unknown names).
   - Prompt building + response parsing are pure functions → unit-tested; HTTP mocked.
2. **Storage** – migration `0007_llm_suggestions.sql`: `llm_suggestions(transaction_id PK, model, input_hash, category, tags_json, created_at)`. `input_hash` = hash(tx text, model, category list, tag list) so new categories invalidate it.
3. **Trigger**: only when top k-NN category score < `fallback_threshold` (or no estimation).
   - Batch: `fin classify` runs it for unreviewed low-confidence transactions after embeddings.
   - TUI: use cached suggestion if hash matches, else async `tea.Cmd` → `llmSuggestionMsg` → `ReviewModel.SetLLMSuggestion`.
4. **Display**: LLM-suggested category/tags are moved to the top of their lists and marked (desc `✦ LLM suggestion`, plus k-NN % if any). Not auto-preselected (no calibrated score); user confirms with one keypress. Status line shows "LLM thinking…" / "LLM unavailable".

## Nix / tooling
Ollama is an external HTTP service for `fin`; no runtime dependency in `nix/package.nix`. Models are pulled at runtime into `~/.ollama/models` (not Nix-managed).
- `flake.nix` devShell: add `ollama` to `packages`.
- `flake.nix` app `apps.ollama-serve` (new `nix/ollama-serve.nix`, `pkgs.writeShellApplication`, `runtimeInputs = [ ollama curl ]`):
  - If a server already answers on `$OLLAMA_HOST` (default `127.0.0.1:11434`, e.g. the user's nix-profile server or `services.ollama`), skip starting one.
  - Else start `ollama serve` in background, wait until `/api/version` responds, `trap` to stop it on exit.
  - `ollama pull` each model given as args, defaulting to `qwen3-embedding:0.6b qwen3:1.7b`.
  - If it started the server, `wait` on it (foreground until Ctrl-C).
- `justfile`: `ollama-serve: nix run .#ollama-serve` (passes through extra model args).
- Document in `categorization.md` (config section): `nix run .#ollama-serve` for dev; NixOS alternative `services.ollama = { enable = true; loadModels = [ ... ]; }`.
- All tests use `httptest`; `nix flake check` needs no network or models.

## Verification
- `nix develop -c just lint`, `just build`, `just test` after each phase; `nix flake check` at the end.
- `nix run .#ollama-serve` with the existing server running → only pulls; with it stopped → starts, pulls, serves.
- Manual: set `[llm] enabled = true`, run `fin classify` on a real workspace, then `fin tui`:
  - k-NN scores appear immediately; scores update once embedding arrives for a new tx.
  - A payee with no token overlap (e.g. new supermarket) gets a sensible top category.
  - Low-confidence tx shows `✦ LLM suggestion` entries from existing labels only.
  - Stop Ollama → TUI and import behave exactly as before, with a warning only.
- Phase 1 tuning: compare top-1 category accuracy Jaccard vs embeddings by leave-one-out over reviewed transactions (small throwaway test/command) to pick `floor`.
