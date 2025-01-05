{
  perSystem =
    {
      config,
      pkgs,
      ...
    }:
    {
      packages.docker = pkgs.dockerTools.buildLayeredImage {
        name = "kalbasit/signal-api-receiver";
        contents = [
          # required for TLS certificate validation
          pkgs.cacert

          # required for working with timezones
          pkgs.tzdata

          # the signal-api-receiver package
          config.packages.signal-api-receiver
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
      };

      packages.push-docker-image = pkgs.writeShellScript "push-docker-image" ''
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
