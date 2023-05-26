{ config, lib, pkgs, ... }:
let cfg = config.services.systemd-networkd-exporter;
in {
  options = {
    services.systemd-networkd-exporter = {
      enable = lib.mkEnableOption "systemd-networkd-exporter";

      listenAddress = lib.mkOption {
        type = lib.types.str;
        default = "0.0.0.0";
        description = lib.mdDoc ''
          Address to listen on for the HTTP server.
        '';
      };

      port = lib.mkOption {
        type = lib.types.port;
        default = 15694;
        description = lib.mdDoc ''
          Port to listen on.
        '';
      };

      dbusAddr = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = lib.mdDoc ''
          Address of the DBus daemon.
        '';
      };
    };
  };

  config = {
    nixpkgs.overlays = [ (import ./overlay.nix) ];

    systemd.services.systemd-networkd-exporter = lib.mkIf cfg.enable {
      description = "Prometheus exporter for systemd-networkd";
      unitConfig = { Type = "simple"; };
      serviceConfig = {
        ExecStart =
          "${pkgs.systemd-networkd-exporter}/bin/systemd-networkd-exporter --logtostderr --listen-address=${cfg.listenAddress}:${
            builtins.toString cfg.port
          } --dbus-addr=${cfg.dbusAddr}";
      };
      wantedBy = [ "multi-user.target" ];
    };
  };
}
