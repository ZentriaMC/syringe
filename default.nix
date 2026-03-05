{ lib
, buildGoModule
, coreutils
, rev ? "dirty"
}:

buildGoModule {
  pname = "syringe";
  version = rev;

  src = lib.cleanSource ./.;

  vendorHash = "sha256-kCalVW5wAEcO+R149xGxuspDR7iywww2xD9vLFuBkeg=";
  subPackages = [ "cmd/syringe" ];
  ldflags = [
    "-s"
    "-w"
    "-X github.com/ZentriaMC/syringe/internal/version.Version=${rev}"
  ];

  postInstall = ''
    install -D -m 644 ./dbus/ee.zentria.syringe1.Syringe.conf $out/share/dbus-1/system.d/ee.zentria.syringe1.Syringe.conf
    install -D -m 644 ./dbus/ee.zentria.syringe1.Syringe.service $out/share/dbus-1/system-services/ee.zentria.syringe1.Syringe.service

    substituteInPlace $out/share/dbus-1/system-services/ee.zentria.syringe1.Syringe.service \
      --replace-fail /bin/false "${coreutils}/bin/false"
  '';

  meta = with lib; {
    description = "systemd credential service implementation";
    homepage = "https://github.com/ZentriaMC/syringe";
    license = licenses.gpl3;
    platforms = platforms.linux;
  };
}
