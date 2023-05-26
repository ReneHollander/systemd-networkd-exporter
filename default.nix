{ lib, buildGoModule }:
buildGoModule rec {
  pname = "systemd-networkd-exporter";
  version = "1.0.0";
  src = lib.cleanSource ./.;
  vendorSha256 = "sha256-KUoDikIjutvnaR794hzkuKeun51vak9nAdTGsRu3BYg=";
  ldflags = [ "-w" "-s" "-X main.version=v${version}" ];
  meta = with lib; { supportedPlatforms = platforms.linux; };
}
