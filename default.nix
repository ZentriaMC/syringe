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

  vendorHash = "sha256-H5fc5RD4d22BhRtr1wZmFcObXsri3T1ttYQLCRX4080=";
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
