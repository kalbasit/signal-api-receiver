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

          ldflags = [
            "-X github.com/kalbasit/signal-api-receiver/cmd.Version=${version}"
          ];

          subPackages = [ "." ];

          vendorHash = "sha256-Q/8AD0d2pNBB/uarwHq+jimDxtg+YfId9XCXprMcU3g=";

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
