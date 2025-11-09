{
  description = "MCMS SDK Flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    chainlink-ton.url = "github:smartcontractkit/chainlink-ton?ref=feat/mcms-polish";
  };

  outputs = inputs @ {
    self,
    nixpkgs,
    flake-utils,
    chainlink-ton,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      # Import nixpkgs with specific configuration
      pkgs = import nixpkgs {
        inherit system;
      };

      # The rev (git commit hash) of the current flake
      rev = self.rev or self.dirtyRev or "-";

      pkgsContracts = {
        chainlink-ton-contracts = chainlink-ton.packages.${system}.contracts;
      };
    in rec {
      # Output a set of dev environments (shells)
      devShells = {
        default = pkgs.callPackage ./shell.nix {inherit pkgs pkgsContracts;};
      };
    });
}
