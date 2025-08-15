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
          shortRev = self.shortRev or self.dirtyShortRev;
          rev = self.rev or self.dirtyRev;
          tag = builtins.getEnv "RELEASE_VERSION";

          version = if tag != "" then tag else rev;
        in
        pkgs.buildGoModule {
          name = "signal-api-receiver-${shortRev}";

          src = lib.fileset.toSource {
            fileset = lib.fileset.unions [
              ../../cmd
              ../../go.mod
              ../../go.sum
              ../../main.go
              ../../pkg
            ];
            root = ../..;
          };

          CGO_ENABLED = 0;

          ldflags = [
            "-X github.com/kalbasit/signal-api-receiver/cmd.Version=${version}"
          ];

          subPackages = [ "." ];

          vendorHash = "sha256-pJR3jbX/6k3iQvBQTtMUWS+0+TamjVP+hcqOTItsEjI=";

          doCheck = true;

          meta = {
            description = "Signal API receiver";
            homepage = "https://github.com/kalbasit/signal-api-receiver";
            license = lib.licenses.mit;
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };

    };
}
