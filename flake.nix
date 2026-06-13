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

        # Main application package
        fin_man = pkgs.callPackage ./nix/package.nix { };

        pre-commit-check = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            # Custom hooks that use justfile recipes for parity
            just-lint = {
              enable = true;
              name = "just-lint";
              entry = let
                path = pkgs.lib.makeBinPath [ pkgs.just pkgs.golangci-lint pkgs.go pkgs.bash pkgs.coreutils ];
              in
              "${pkgs.bash}/bin/sh -c '
                export PATH=${path}:$PATH
                export GOCACHE=$TMPDIR/gocache
                ln -snf ${fin_man.goModules} vendor
                export GOFLAGS=-mod=vendor
                just lint
              '";
              files = "\\.go$";
              pass_filenames = false;
            };

            # Standard hooks
            trim-trailing-whitespace.enable = true;
            end-of-file-fixer.enable = true;
            check-yaml.enable = true;
            check-added-large-files.enable = true;
          };
        };
      in
      {
        packages = {
          inherit fin_man;
          default = fin_man;
        };

        checks = {
          inherit pre-commit-check;
        };

        devShells.default = pkgs.mkShell {
          inherit (pre-commit-check) shellHook;

          inputsFrom = [ fin_man ];

          packages = with pkgs; [
            # Development tools
            delve
            golangci-lint
            gopls
            gotools

            # Database tools
            goose
            sqlc
            sqlite

            # Task runner & misc
            just
            git
          ];

          buildInputs = pre-commit-check.enabledPackages;
        };
      });
}
