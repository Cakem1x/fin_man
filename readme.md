# Finance Manager

[![CI](https://github.com/Cakem1x/fin_man/actions/workflows/ci.yml/badge.svg)](https://github.com/Cakem1x/fin_man/actions/workflows/ci.yml)

I'm trying to build a bunch of CLI tools to manage personal finances.
WIP.

## Development

### Nix: Update vendor hash
When updating Go dependencies, you must update the `vendorHash` in `nix/package.nix`:
1. Change `vendorHash = ""` in `nix/package.nix`.
2. Run `nix build .#fin_man` which will fail and print the correct hash.
3. Copy the `got: sha256-...` hash from the error and paste it as the new `vendorHash`.
