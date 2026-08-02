# Agent Instructions for fin_man

## Environment & Tooling
This project uses `nix flake` and `just` for its development environment and task running.

When you need to run tests, build, or lint the code, you **MUST** run the commands defined in the `justfile` inside the Nix development shell.

- **Run all checks (Lint, Build, Test)**: `nix develop -c just ci`
- **Test**: `nix develop -c just test`
- **Lint**: `nix develop -c just lint`
- **Build**: `nix develop -c just build`

If you are already inside a `direnv` environment where `use flake` is active, you can simply run the `just <recipe>` commands directly.
