{
  lib,
  buildGoModule,
  installShellFiles,
  makeWrapper,
  gocryptfs,
}:

buildGoModule {
  pname = "fin_man";
  version = "0.1.0";
  src = ../.; # Root of the repository
  vendorHash = "sha256-7Sqo3TJNhdS7Tt8x63sVviBrYQaYX6Ah+3g+29vVszI=";

  subPackages = [ "cmd/fin" ];

  nativeBuildInputs = [ installShellFiles makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/fin \
      --prefix PATH : ${lib.makeBinPath [ gocryptfs ]}
  '';

  meta = with lib; {
    description = "Personal finance CLI suite";
    homepage = "https://github.com/cakemix/fin_man";
    license = licenses.mit; # Based on LICENSE file
    maintainers = with maintainers; [ ];
    mainProgram = "fin";
  };
}
