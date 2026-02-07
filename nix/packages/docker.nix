{
  perSystem =
    {
      config,
      lib,
      pkgs,
      ...
    }:
    {
      packages.docker = pkgs.dockerTools.buildLayeredImage {
        name = "kalbasit/signal-api-receiver";
        contents =
          let
            etc-passwd = pkgs.writeTextFile {
              name = "passwd";
              text = ''
                root:x:0:0:Super User:/homeless-shelter:/dev/null
                ncps:x:1000:1000:NCPS:/homeless-shelter:/dev/null
              '';
              destination = "/etc/passwd";
            };

            etc-group = pkgs.writeTextFile {
              name = "group";
              text = ''
                root:x:0:
                ncps:x:1000:
              '';
              destination = "/etc/group";
            };
          in
          [
            # required for Open-Telemetry auto-detection of process information
            etc-passwd
            etc-group

            # required for TLS certificate validation
            pkgs.cacert

            # required for working with timezones
            pkgs.tzdata

            # the signal-api-receiver package
            (config.packages.signal-api-receiver.overrideAttrs (oa: {
              # Disable tests for the docker image build. Also remove the
              # coverage output that's only generated when tests run. This is
              # because they provide no value in this package since the default
              # package (signal-api-receiver) of the flake already runs the
              # tests.
              doCheck = false;
              outputs = lib.remove "coverage" (oa.outputs or [ ]);
            }))
          ];
        config = {
          Cmd = [
            "/bin/signal-api-receiver"
            "serve"
          ];
          ExposedPorts = {
            "8105/tcp" = { };
          };
          Labels = {
            "org.opencontainers.image.description" = "Signal API Receiver";
            "org.opencontainers.image.licenses" = "MIT";
            "org.opencontainers.image.source" = "https://github.com/kalbasit/signal-api-receiver";
            "org.opencontainers.image.title" = "signal-api-receiver";
            "org.opencontainers.image.url" = "https://github.com/kalbasit/signal-api-receiver";
          };
        };

        fakeRootCommands = ''
          #!${pkgs.runtimeShell}
          mkdir -p tmp
          chmod -R 1777 tmp
        '';
      };

      packages.push-docker-image = pkgs.writeShellScriptBin "push-docker-image" ''
        set -euo pipefail

        if [[ ! -v DOCKER_IMAGE_TAGS ]]; then
          echo "DOCKER_IMAGE_TAGS is not set but is required." >&2
          exit 1
        fi

        for tag in $DOCKER_IMAGE_TAGS; do
          echo "Pushing the image tag $tag for system ${pkgs.hostPlatform.system}. final tag: $tag-${pkgs.hostPlatform.system}"
          ${pkgs.skopeo}/bin/skopeo --insecure-policy copy \
            "docker-archive:${config.packages.docker}" docker://$tag-${pkgs.hostPlatform.system}
        done
      '';
    };
}
