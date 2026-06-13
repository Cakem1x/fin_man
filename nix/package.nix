{
  lib,
  buildGoModule,
  installShellFiles,
}:

buildGoModule {
  pname = "fin_man";
  version = "0.1.0";
  src = ../.; # Root of the repository
  vendorHash = "sha256-ctSwUpgKXwwS4YZIfsbg3dk7eOqRY+BUcu8GaqZ2hpI=";

  subPackages = [ "cmd/fin" ];

  nativeBuildInputs = [ installShellFiles ];

  meta = with lib; {
    description = "Personal finance CLI suite";
    homepage = "https://github.com/cakemix/fin_man";
    license = licenses.mit; # Based on LICENSE file
    maintainers = with maintainers; [ ];
    mainProgram = "fin";
  };
}
