{
  perSystem =
    {
      self',
      pkgs,
      config,
      ...
    }:
    {
      checks =
        self'.packages
        // self'.devShells
        // {
          golangci-lint-check = config.packages.signal-api-receiver.overrideAttrs (oa: {
            name = "golangci-lint-check";
            src = ../../.;
            # ensure the output is only out since it's the only thing this package does.
            outputs = [ "out" ];
            nativeBuildInputs = oa.nativeBuildInputs ++ [ pkgs.golangci-lint ];
            buildPhase = ''
              HOME=$TMPDIR
              golangci-lint run --timeout 10m
            '';
            installPhase = ''
              touch $out
            '';
            doCheck = false;
          });
        };
    };
}
