# Generate real IDs for categories and tags

## Context
`EnrichTransaction` in `internal/db/db.go` uses the category name and tag name as the ID (see the long comment there).
After `RenameCategory`, the ID still equals the old name. The estimator's online learning in `OverviewModel.saveReview` also uses the name as the ID.
The coupling of name and ID is hidden and confusing.

## Scope
- New categories and tags get a random ID: 16 bytes from `crypto/rand`, hex-encoded. No new dependency.
- Existing IDs stay valid. IDs are opaque strings; no migration is needed.
- Remove the comment block about name IDs in `EnrichTransaction`.
- Online learning uses the real category and tag IDs (read them back after saving).

## Acceptance criteria
Write failing tests first:
- `EnrichTransaction` with a new category creates a category whose ID differs from its name.
- Enriching two transactions with the same new tag name links both to the same tag ID.

## Depends on
None.
