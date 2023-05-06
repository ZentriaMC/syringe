let
  flake = builtins.getFlake (toString ../.);

  pkgs = import flake.inputs.nixpkgs {
    system = builtins.currentSystem;
  };

  nixos-lib = import "${flake.inputs.nixpkgs}/nixos/lib" { };
in
import ./test.nix {
  inherit (nixos-lib) runTest;
  inherit flake;
  hostPkgs = pkgs;
}
