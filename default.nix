{ lib
, buildGoModule
, fetchFromGitHub
, coreutils
, rev ? "dirty"
}:

buildGoModule rec {
  pname = "syringe";
  version = rev;

  src = lib.cleanSource ./.;

  vendorHash = "sha256-BYa1t6NwtjiHdAqGQdSX15D1FnSLaI7bv78S6F8G/NM=";
  subPackages = [ "cmd/syringe" ];
  ldflags = [
    "-s"
    "-w"
    "-X github.com/ZentriaMC/syringe/internal/version.Version=${version}"
  ];

  postInstall = ''
    install -D -m 644 ./dbus/ee.zentria.syringe1.Syringe.conf $out/share/dbus-1/system.d/ee.zentria.syringe1.Syringe.conf
    install -D -m 644 ./dbus/ee.zentria.syringe1.Syringe.service $out/share/dbus-1/system-services/ee.zentria.syringe1.Syringe.service

    substituteInPlace $out/share/dbus-1/system-services/ee.zentria.syringe1.Syringe.service \
      --replace /bin/false "${coreutils}/bin/false"
  '';

  meta = with lib; {
    description = "systemd credential service implementation";
    homepage = "https://github.com/ZentriaMC/syringe";
    license = licenses.gpl3;
    platforms = platforms.linux;
  };
}
