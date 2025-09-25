{
  stdenv,
  pkgs,
  lib,
}:
pkgs.mkShell {
  buildInputs = with pkgs;
    [
      # nix tooling
      alejandra

      # Go 1.24 + tools
      go_1_24
      gopls
      delve
      golangci-lint
      gotools
      go-mockery
      go-task # taskfile runner

      # Rust + tools
      # rustc
      # cargo
      # solana-cli

      # TS/Node set of tools for changesets
      nodejs_24
      (pnpm.override {nodejs = nodejs_24;})
      nodePackages.typescript
      nodePackages.typescript-language-server
      # Required dependency for @ledgerhq/hw-transport-node-hid -> usb
      nodePackages.node-gyp

      # Extra tools
      git
      jq
      yq-go # for manipulating golangci-lint config
    ]
    ++ lib.optionals stdenv.hostPlatform.isDarwin [
      libiconv
    ];
}
