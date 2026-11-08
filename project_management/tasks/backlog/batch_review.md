# Batch review of unreviewed transactions

## Context
`fin tui` reviews one transaction at a time. This works for fixing one or two transactions.
After an import of several hundred transactions, it is slow: each transaction needs its own edit view, even when the suggestion is obviously correct.

Goal: a separate subcommand, `fin review`, that only processes unreviewed transactions.
It preselects the most likely category and tags for every transaction and shows how certain each suggestion is.
The user confirms certain suggestions in bulk and spends time only on uncertain ones.

The principle "the user confirms every categorization" (`architecture/overview.md`) still holds.
Bulk confirmation is an explicit user action; nothing is marked reviewed automatically.

Follow AGENTS.md: TDD (failing test first), `just lint/build/test`, `nix flake check` at the end.
Large task: ask whether to run it in a herdr worktree. Consider splitting it into subtasks along the phases below.

## Decisions
- `fin review` is a Bubble Tea TUI. A table fits the workflow (scan many rows, act on many rows) better than a line-by-line prompt.
- Suggestions come from `app.Service` (`Suggest`). The batch view adds no own estimator logic.
- Each confirmed transaction is persisted immediately with `ApplyReview` (`reviewed = true`). This trains the estimator online, so later suggestions improve during the session. A crash loses no work.
- `u` undoes the last action (restores the previous category, tags, and reviewed state of all affected transactions). Only the last action; no undo history.
- `fin review` shows only unreviewed transactions. Reviewed transactions are inspected and fixed in `fin tui`.
- `fin import-csv` prints a hint at the end, e.g. `N unreviewed transactions. Run 'fin review' to review them.` It does not start `fin review`.
- For complex cases (split, receipt matcher, note), the user opens the existing edit view for that transaction. The batch view does not duplicate it.

## Confidence levels
New pure function in `internal/categorize` (or `internal/app`), e.g. `Confidence(est *Estimation) Level`, with levels:

- `Certain`: one category with a high score and no close competitor, and every tag score is clearly high or clearly low.
  Indicative rule: top category score ≥ 0.9, second category score ≤ 0.1, no tag score in `(0.2, 0.8)`.
- `Likely`: top category score > 0.8 (the current preselect threshold), but not `Certain`.
- `Unsure`: top category score ≤ 0.8 or several categories with similar scores.
- `None`: no estimation (no similar reviewed transaction).

The thresholds are constants. Pick them with a leave-one-out run over reviewed transactions: the `Certain` level must reach a top-1 accuracy of ~98% or more.
This can share the throwaway evaluation planned in `local_llm_classification.md` (Phase 1 tuning).

Document the levels and thresholds in `architecture/categorization.md` (single source of truth for the algorithm).

## Phase 1: batch view without generative LLM
1. **Confidence**: `Confidence` function as above. Unit tests per level.
2. **Command**: `cmd/fin/cmd/review.go` (`fin review`). Opens the workspace with the shared helper (`workspace_db.go`) and builds `app.Service` like `tui.go`.
3. **Table**: one row per unreviewed transaction: date, payee, amount, preselected category, preselected tags, confidence marker.
   - Markers use symbol and color, so they work without color: e.g. `●` green `Certain`, `◐` yellow `Likely`, `○` red `Unsure`, `·` grey `None`.
   - The score of the preselected category is shown as a percentage.
   - Default sort: by confidence (`Certain` first), then by payee, so similar transactions are adjacent.
   - Filter by confidence level.
4. **Actions** (keymap and help line per `tui_keymap.md`):
   - Confirm the focused row.
   - Mark rows (single, range, all rows of a confidence level).
   - Confirm all marked rows. Show a confirmation dialog with the count and a short summary per category.
   - Cycle the category of the focused row through the next-best suggestions.
   - Open the category picker or tag picker for all marked rows (sets the same value for all).
   - Group: mark all rows with the same normalized payee (`categorize.EmbeddingText` or the tokenizer) as the focused row.
   - Open the edit view of `fin tui` for the focused row.
   - Undo.
5. **Re-estimation**: after each persisted action, re-estimate the remaining rows and update their preselection and confidence. Rows the user has edited in this session keep the user's values.
   Re-estimation of all rows must stay fast for ~1000 unreviewed transactions; measure it.
6. **Progress**: status line with reviewed/remaining counts per confidence level.
7. **Import hint**: `fin import-csv` prints the number of unreviewed transactions and the `fin review` hint if the number is greater than zero.

## Phase 2: generative LLM for unsure rows
Depends on Phase 2 of `local_llm_classification.md` (`LLMClassifier`, `llm_suggestions` cache). Do not reimplement it here.

1. On start, if `[llm]` is enabled, classify all `Unsure` and `None` rows in the background with bounded concurrency. Use cached suggestions if the hash matches. The table stays usable; rows update when their result arrives.
2. Combine with the k-NN result:
   - LLM agrees with the k-NN top category: raise `Unsure` to `Likely`. Never to `Certain`.
   - LLM disagrees: keep `Unsure`, show the LLM category as an alternative with the `✦` marker.
   - No k-NN estimation: preselect the LLM category, level stays `None` with the `✦` marker.
3. Status line: number of pending LLM requests, or "LLM unavailable". If Ollama is unreachable, the view works as in Phase 1.

## Out of scope
- Automatic review without user confirmation.
- Showing or editing reviewed transactions in `fin review`.
- Undo history beyond the last action.
- Starting `fin review` automatically from `fin import-csv`.
- Changes to the edit view of `fin tui` (`tui_edit_view.md`).

## Acceptance criteria
Write failing tests first:
- `Confidence` returns the expected level for: a single dominant category with clear tags; a dominant category with one ambiguous tag; two close categories; no estimation.
- Confirming all marked `Certain` rows persists their preselected category and tags with `reviewed = true`, and leaves unmarked rows unreviewed.
- After confirming a row, a similar unreviewed row gets the confirmed category as top suggestion.
- Undo after a bulk confirmation restores the previous state of all affected transactions.
- A row edited by the user keeps its values after re-estimation.
- Phase 2 (LLM mocked with `httptest`): agreement raises `Unsure` to `Likely`; disagreement keeps `Unsure` and offers the LLM category.

## Docs
- `architecture/overview.md`: add `review` to the main components.
- `architecture/categorization.md`: confidence levels, batch review workflow, LLM combination rules.
- `architecture/roadmap.md`: tick "batch review".

## Depends on
- `app_service_layer.md` (done on its branch, not yet merged).
- `review_draft_model.md`: the batch view stores per-row edits as `app.Draft`.
- `tui_keymap.md`: keymap and help line conventions.
- Phase 2: `local_llm_classification.md` Phase 2.
