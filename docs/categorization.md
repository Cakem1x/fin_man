# Categorization

## Pipeline

1. deterministic rules
2. similarity suggestions
3. manual review

## Rule Examples

- payee contains
- regex match
- IBAN match
- account-specific rules

## Similarity

Use lightweight local heuristics:

- trigram similarity
- token overlap
- normalized merchant names

Avoid ML initially.

## Review Workflow

The user reviews proposed categories in a TUI and can:

- accept
- edit
- create rules
- add tags
