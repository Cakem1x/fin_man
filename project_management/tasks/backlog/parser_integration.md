# Backlog: External Parser Integration

## Proposal: `bankstatementparser` (Python)
Instead of rolling custom parsers for complex formats (PDF, OFX, MT940), we can integrate [bankstatementparser](https://github.com/sebastienrousseau/bankstatementparser).

### Status: Not Taken
We are currently sticking to native Go parsers for performance and simplicity. This option should be revisited if parsing requirements (e.g., scanned PDFs or complex banking XMLs) exceed what is reasonable to maintain in Go.

### Integration Strategy (Option 1: CLI Shell)
- **Mechanism**: Implement a `TransactionImporter` that executes the `bankstatementparser` CLI via `os/exec` and parses its JSON output.
- **Dependency Management**: Ensure Python and the required package are available in the development environment via the project's `flake.nix`.

### Benefits
- Instant support for diverse banking formats.
- LLM/Vision fallback for digital and scanned PDFs.
- Built-in transaction deduplication and balance verification.

### Risks
- Introduces a Python runtime dependency.
- Higher overhead compared to native Go implementation.
