{
  stdenv,
  pkgs,
  lib,
  # smart contract pkgs to load
  pkgsContracts,
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
      go-mockery_2
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
      go-task
    ]
    ++ builtins.attrValues pkgsContracts
    ++ lib.optionals stdenv.hostPlatform.isDarwin [
      libiconv

      # Required to support go build inside a nix devshell (c compiler dependency on SecTrustCopyCertificateChain/macOS 12+)
      # https://github.com/NixOS/nixpkgs/issues/433688#issuecomment-3231551949
      pkgs.apple-sdk_15
    ];

  PATH_CONTRACTS_TON = "${pkgsContracts.chainlink-ton-contracts}/lib/node_modules/@chainlink/contracts-ton/build/";

  shellHook = ''
    echo "TON contracts located here: $PATH_CONTRACTS_TON"
  '';
}
