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

## How to Integrate Tests Into Development
### Which Tests to add
Add concise tests for features and regression tests for bugs, when possible.
Avoid adding tests that lock in implementation details.
Only things that are essential for the program's logic or part of the intended contract of a function shall be tested.
One exception: Using tests to drive the development workflow of complex implementation details is good practice, but keep these tests separate from those that test the 'public' interface.

### How to add Tests
Whenever possible, start by adding tests and verify that they fail.
Then, adapt code and check that they pass.
This helps ensuring tests actually test what you want to test.
