# Agent Instructions for fin_man

## Environment & Tooling
This project uses `nix flake` and `just` for its development environment and task running.

Use these commands to validate your changes.
- **Lint**: `nix develop -c just lint`
- **Build**: `nix develop -c just build`
- **Test**: `nix develop -c just test`

Final test, use sparsely. Only before finishing the task:
- `nix flake check`

If you are already inside a `direnv` environment where `use flake` is active, you can simply run the `just <recipe>` commands directly.

## Documentation
When adding comments or docs, ensure to use technical writing best practices.
- Short, concise sentences
- Exactly the same term to the same thing, always
- No marketing

Generally, avoid adding superfluous documentation.
Before adding comments, consider whether refactoring could replace the need for a comment.

Make sure that a thing is documented at one location only, to avoid running into inconsistent documentation later.

## Tests

### How to add Tests: Test driven development (TDD)
Do not add tests that succeed after implementation is done.
Instead, tests need to be added beforehand and need to fail.
This verifies that the test is correctly testing what we want it to test

### What to test
100% coverage is not a goal. Use tests as a tool to
- drive development (TDD)
- and to document and protect from regressions important structure / interfaces

Do not add tests for implementation details of a function.
If logic is sufficiently complex to warrent tests (TDD), move that functionality into a separate function that can be tested properly.
