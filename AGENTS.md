# Agent Instructions for fin_man

## Worktree and Tasks
- The default branch is 'main'. Merging to main is done by the user.
- When instructed, create a new worktree to run your task in. If you get a large task, ask whether it should run in a separate worktree.
- When running in herdr (`HERDR_ENV=1`), use worktree-based workflows. Use `herdr worktree create` to create both the git worktree and a herdr workspace in one step.
  Worktrees are siblings of the `main` checkout (e.g. `fin_man/feat-add-login-page` next to `fin_man/main`), never in herdr's default directory:
  ```bash
  root="$(dirname "$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")")"
  resp="$(herdr worktree create --cwd "$PWD" --branch <branch-name> --path "$root/<dir-name>" --label "<task label>" --no-focus)"
  wt="$(jq -r '.result.worktree.path' <<<"$resp")"
  pane="$(jq -r '.result.root_pane.pane_id' <<<"$resp")"
  ```
  Derive `<branch-name>` from the task description (e.g., `feat/add-login-page`). `<dir-name>` is the branch name with `/` replaced by `-`.
  Use `$wt` as the working directory for all task commands.
- The task runs in its own agent in the new herdr workspace, not as a subagent of the calling session:
  1. In the worktree, move the task file from `project_management/tasks/backlog/` to `project_management/tasks/active/` with `git mv` and commit.
     The fresh worktree has no `.pre-commit-config.yaml` until `nix develop` runs there, so commit with `PRE_COMMIT_ALLOW_NO_CONFIG=1`.
  2. Start the agent in the root pane. Use the task file name as the agent name:
     ```bash
     herdr agent start <task-name> --kind claude --pane "$pane"
     ```
     If it returns `agent_not_ready`, read the pane (`herdr agent read <task-name> --source visible`). A Claude Code folder trust prompt or any other approval dialog is answered by the user, never by the calling agent.
  3. Send the task prompt without `--wait`, so the calling session is not blocked:
     ```bash
     herdr agent prompt <task-name> "Work on the task in project_management/tasks/active/<task-name>.md. Follow AGENTS.md."
     ```
  Check progress with `herdr agent get <task-name>` and `herdr agent read <task-name> --source recent-unwrapped`.
- When in a worktree task, commit changes. When in `main`, do not commit changes automatically.
- Read the task file from `project_management/tasks/active/` for requirements and context. Reference architecture docs in `project_management/architecture/` as needed.
- When the task is done, move the task file to `project_management/tasks/done/` with `git mv` and commit.

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
