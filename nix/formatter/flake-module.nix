{ inputs, ... }:
{
  imports = [ inputs.treefmt-nix.flakeModule ];

  perSystem = {
    treefmt = {
      # Used to find the project root
      projectRootFile = ".git/config";

      settings.global.excludes = [
        ".envrc"
        "LICENSE"
        "renovate.json"
      ];

      programs = {
        actionlint.enable = true;
        deadnix.enable = true;
        gofumpt.enable = true;
        mdformat.enable = true;
        nixfmt.enable = true;
        statix.enable = true;
        yamlfmt.enable = true;
      };
    };
  };
}
