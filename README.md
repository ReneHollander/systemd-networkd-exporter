# systemd-networkd-exporter

Prometheus exporter for systemd-networkd

# Supported metrics

- [x] DHCP Leases (also supports usage of DHCP Pool info available through JSON and follow up PR to expose the hostname: https://github.com/systemd/systemd/pull/27465)

That's it for now...

# Installation

## Nix flake

In your `flake.nix`:
```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    systemd-networkd-exporter.url = "github:ReneHollander/systemd-networkd-exporter";

    systemd-networkd-exporter.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, ... }@inputs: {
    nixosConfigurations."hostname" = nixpkgs.lib.nixosSystem rec {
      system = "x86_64-linux";
      modules = [
        inputs.systemd-networkd-exporter.nixosModule
      ];
    };
  };
}
```

In your configuration:
```nix
  imports = [
    ...
    inputs.systemd-networkd-exporter.nixosModule
    ...
  ];

  services.systemd-networkd-exporter.enable = true;
```
