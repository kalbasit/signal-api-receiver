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
            "-X github.com/leptoquark1/signal-api-receiver/cmd.Version=${version}"
          ];

          subPackages = [ "." ];

          vendorHash = "sha256-Uo2/bkBPHatbNCC90U3uKZ2n5XFFpO4dM/M1EGg0HRE=";

          doCheck = true;

          meta = {
            description = "Signal API receiver";
            homepage = "https://github.com/leptoquark1/signal-api-receiver";
            license = lib.licenses.mit;
            maintainers = [ lib.maintainers.leptoquark1 ];
          };
        };

    };
}
