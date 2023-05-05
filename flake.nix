{
  description = "Syringe";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      rev = self.rev or "dirty";

      supportedSystems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
    in
    {
      nixosModules.default = self.nixosModules.syringe;
      nixosModules.syringe = import ./nixos/module.nix;

      overlays.default = self.overlays.syringe;
      overlays.syringe = final: prev: {
        syringe = prev.callPackage ./default.nix {
          inherit rev;
        };
      };
    } // flake-utils.lib.eachSystem supportedSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        inherit (pkgs) lib stdenv;
      in
      rec {
        packages.default = packages.syringe;
        packages.syringe = pkgs.callPackage ./default.nix {
          inherit rev;
        };

        devShell = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.golangci-lint
            pkgs.gopls
          ] ++ lib.optionals stdenv.isDarwin [
            pkgs.podman
            pkgs.qemu
          ];
        };
      });
}
