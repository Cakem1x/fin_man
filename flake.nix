{
  description = "Personal finance CLI suite";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            # Go toolchain
            go
            gopls
            delve

            # Database tools
            sqlite
            goose

            # SQL/codegen
            sqlc

            # Build/test helpers
            just
            git
            gnumake

            # Linting/formatting
            golangci-lint
            gotools

            # Optional: encryption/sync helpers
            #age
            #restic
            #gocryptfs
          ];
        };
      });
}
