{ self, ... }:
{
  perSystem =
    {
      lib,
      pkgs,
      ...
    }:
    {
      packages.signal-api-receiver =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;

          vendorHash = "sha256-6jvEKTmH9MexNHQZwUGpNFpLKpw6fYbylgzk9cG2kcI=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          name = "signal-api-receiver";

          src = lib.fileset.toSource {
            fileset = lib.fileset.unions [
              ../../../cmd
              ../../../go.mod
              ../../../go.sum
              ../../../main.go
              ../../../pkg
            ];
            root = ../../..;
          };

          ldflags = [
            "-X github.com/kalbasit/signal-api-receiver/cmd.Version=${version}"
          ];

          subPackages = [ "." ];

          doCheck = true;

          checkFlags = [
            "-race"
            "-coverprofile=coverage.txt"
          ];

          outputs = [
            "out"
            "coverage"
          ];

          postCheck = ''
            mv coverage.txt $coverage
          '';

          meta = {
            description = "Signal API receiver";
            homepage = "https://github.com/kalbasit/signal-api-receiver";
            license = lib.licenses.mit;
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };

    };
}
