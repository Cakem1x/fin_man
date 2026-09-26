# Agent Instructions for fin_man

## Environment & Tooling
This project uses `nix flake` and `just` for its development environment and task running.

Use these commands to validate your changes.
- **Lint**: `nix develop -c just lint`
- **Build**: `nix develop -c just build`
- **Test**: `nix develop -c just test`
- **Use sparsely, only before finishing the task: Run all checks (Lint, Build, Test)**: `nix develop -c just ci`

If you are already inside a `direnv` environment where `use flake` is active, you can simply run the `just <recipe>` commands directly.
