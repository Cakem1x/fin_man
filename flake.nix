{
  description = "Personal finance CLI suite";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks.url = "github:cachix/pre-commit-hooks.nix";
  };

  outputs = { self, nixpkgs, flake-utils, pre-commit-hooks }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        pre-commit-check = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            # Custom hooks that use justfile recipes for parity
            just-lint = {
              enable = true;
              name = "just-lint";
              entry = "just lint";
              files = "\\.go$";
              pass_filenames = false;
            };
          };
        };
      in
      {
        checks = {
          inherit pre-commit-check;
        };

        devShells.default = pkgs.mkShell {
          inherit (pre-commit-check) shellHook;
          buildInputs = pre-commit-check.enabledPackages;

          packages = with pkgs; [
            # Go toolchain
            delve
            go
            golangci-lint
            gopls
            gotools

            # Database tools
            goose # db migration manager
            sqlc # SQL/codegen
            sqlite

            # misc tools
            just
            git
          ];
        };
      });
}
