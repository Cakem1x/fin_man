# Categorization

## Pipeline

1. similarity suggestions
2. manual review

There are no deterministic categorization rules.
The user confirms every categorization.
The estimator learns from each confirmed transaction.

## Similarity Suggestions

The local estimator uses token-set Jaccard similarity over the payee and memo.
For each previously categorized transaction, it raises the similarity to the
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
