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
  vendorHash = "sha256-DgcmFVUo7mLmbz1DKy6oUTmPPT0ZbTxkXBa19ingdkQ=";

  subPackages = [ "cmd/fin" ];

  nativeBuildInputs = [ installShellFiles makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/fin \
      --prefix PATH : ${lib.makeBinPath [ gocryptfs ]}

    installShellCompletion --cmd fin \
      --bash <($out/bin/fin completion bash) \
      --zsh <($out/bin/fin completion zsh) \
      --fish <($out/bin/fin completion fish)
  '';

  meta = with lib; {
    description = "Personal finance CLI suite";
    homepage = "https://github.com/cakemix/fin_man";
    license = licenses.mit; # Based on LICENSE file
    maintainers = with maintainers; [ ];
    mainProgram = "fin";
  };
}
