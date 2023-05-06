{ runTest
, hostPkgs
, flake
}:

runTest {
  name = "syringe-test";

  inherit hostPkgs;

  nodes.machine = {
    imports = [ ./module.nix ];
    nixpkgs.overlays = [
      flake.outputs.overlays.syringe
    ];
  };

  defaults = {
    services.syringe = {
      enable = true;
      supportUpdating = true;

      configText = builtins.readFile ../config.sample.yml;
    };
  };

  testScript = ''
    start_all()

    machine.wait_for_unit("sockets.target")
    machine.succeed("""
      systemd-run -GPdq -u secrets-test --service-type=oneshot --property=LoadCredential="foobarbaz":/run/syringe/syringe.sock /run/current-system/sw/bin/bash -exc 'echo "$(< ''${CREDENTIALS_DIRECTORY}/foobarbaz)"'
    """)
  '';
}
