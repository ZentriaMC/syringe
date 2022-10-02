{ config, lib, pkgs, ... }:

let
  cfg = config.services.syringe;
  suidHelper = "${config.security.wrapperDir}/syringe";

  configPath = "/etc/syringe/config.yml";
in
with lib; {
  options = {
    services.syringe = {
      enable = mkEnableOption "Whether to enable syringe service";
      supportUpdating = mkEnableOption "Whether to enable syringe secret updating support - requires SUID/SGID wrapper";
      socketPaths = mkOption {
        type = types.listOf types.str;
        default = [ "/run/syringe/syringe.sock" ];
        description = mdDoc ''
          A list of unix sockets syringe should listen on. The format follows
          ListenStream as described in systemd.socket(5).
          Note that only unix sockets (ListenStream=) are supported.
        '';
      };

      configText = mkOption {
        type = types.lines;
        default = ''
          ---
          templates: []
        '';
        description = mdDoc ''
          Configuration to be written into config.yml
        '';
      };
    };
  };

  config = mkIf cfg.enable {
    environment.etc."syringe/config.yml" = {
      text = cfg.configText;
    };

    systemd.services.syringe = {
      wantedBy = [ "multi-user.target" ];
      after = [ "syringe.socket" ];
      requires = [ "syringe.socket" ];
      aliases = [ "dbus-ee.zentria.syringe1.Syringe.service" ];

      environment = {
        SYRINGE_SERVER_CONFIG = configPath;
        SYRINGE_SERVER_DBUS = "true";
      };

      serviceConfig = {
        Type = "dbus";
        BusName = "ee.zentria.syringe1.Syringe";
        #DynamicUser = true; # TODO: https://github.com/systemd/systemd/issues/9503
        ExecStart = "${pkgs.syringe}/bin/syringe server";
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
      };
    };

    systemd.sockets.syringe = {
      wantedBy = [ "sockets.target" ];
      socketConfig = {
        ListenStream = cfg.socketPaths;
        SocketMode = "0600";
        SocketUser = "root";
        SocketGroup = "root";
      };
    };

    services.dbus.packages = [ pkgs.syringe ];

    security.wrappers.syringe = mkIf (cfg.enable && cfg.supportUpdating) {
      setuid = true;
      setgid = true;
      owner = "root";
      group = "root";
      source = "${pkgs.syringe}/bin/syringe";
    };
  };
}
