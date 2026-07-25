# Finance Manager

[![CI](https://github.com/Cakem1x/fin_man/actions/workflows/ci.yml/badge.svg)](https://github.com/Cakem1x/fin_man/actions/workflows/ci.yml)

I'm trying to build a bunch of CLI tools to manage personal finances.
WIP.

## Development

### Justfile
You can use the justfile to run common tasks during development, e.g. `just build`.
If you don't have just installed, it's still useful to look of stuff.

Run the linter before committing.

### Nix: Update vendor hash
When updating Go dependencies, you must update the `vendorHash` in `nix/package.nix`:
1. Change `vendorHash = ""` in `nix/package.nix`.
2. Run `nix build .#fin_man` which will fail and print the correct hash.
3. Copy the `got: sha256-...` hash from the error and paste it as the new `vendorHash`.
