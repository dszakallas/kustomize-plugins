{ pkgs, inputs, lib, config, ... }:
{
  options = {
    version = lib.mkOption {
      type = pkgs.lib.types.str;
      default = "develop";
      description = "Version number for the package.";
    };
  };
  config = {
    languages.go.enable = true;

    packages = [ pkgs.git ];

    enterShell = ''
      go version
    '';

    enterTest = ''
      make test
    '';

    outputs = let
      params = { 
        vendorHash = "sha256-9xZrmsJxxqB+Shluy3KtxRQxxJwm3tzhJLMG6KfbJhM=";
        version = config.version;
      };
    in 
      inputs.flake-utils.lib.eachDefaultSystem (system: {
        default = inputs.nixpkgs.legacyPackages.${system}.callPackage ./pkg.nix params;
      });
  };
}