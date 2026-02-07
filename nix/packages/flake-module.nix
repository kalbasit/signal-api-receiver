{
  imports = [
    ./docker.nix
    ./signal-api-receiver
  ];

  perSystem =
    { config, ... }:
    {
      packages.default = config.packages.signal-api-receiver;
    };
}
