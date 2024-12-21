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
        in
        pkgs.buildGoModule {
          name = "signal-api-receiver-${shortRev}";

          src = lib.fileset.toSource {
            fileset = lib.fileset.unions [
              ../../go.mod
              ../../go.sum
              ../../main.go
              ../../receiver
              ../../server
            ];
            root = ../..;
          };

          subPackages = [ "." ];

          vendorHash = "sha256-XlJn0fnFdi9UqJMQHcPPmivHXBbSEMb2trI3MXfyyZ4=";

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
