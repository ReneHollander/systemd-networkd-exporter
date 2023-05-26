{
  description = "Prometheus exporter for systemd-networkd";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }:
    let
      supportedSystems = [ "aarch64-linux" "x86_64-linux" ];
      allSystems = supportedSystems ++ [ "aarch64-darwin" "x86_64-darwin" ];
    in (flake-utils.lib.eachSystem supportedSystems (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in rec {
        packages.systemd-networkd-exporter = pkgs.callPackage ./. { };
        defaultPackage = packages.systemd-networkd-exporter;
      })) // (flake-utils.lib.eachSystem allSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          devShell = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls gotools go-tools ];
          };
        })) // {
          overlays.systemd-networkd-exporter = import ./nixos/overlay.nix;
          overlay = self.overlays.systemd-networkd-exporter;
          nixosModules.systemd-networkd-exporter = import ./nixos/module.nix;
          nixosModule = self.nixosModules.systemd-networkd-exporter;
        };
}
