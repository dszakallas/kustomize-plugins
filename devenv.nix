{
  pkgs,
  inputs,
  lib,
  config,
  ...
}:
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
    env.GOPATH = lib.mkForce null;

    packages = [ pkgs.git ];

    enterShell = ''
      go version
    '';

    enterTest = ''
      make test
    '';

    outputs =
      let
        params = {
          vendorHash = "sha256-9xZrmsJxxqB+Shluy3KtxRQxxJwm3tzhJLMG6KfbJhM=";
          version = config.version;
        };
      in
      inputs.flake-utils.lib.eachDefaultSystem (
        system:
        let
          stdenv = (
            pkgs.stdenv
            // {
              hostPlatform = inputs.nixpkgs.legacyPackages.${system}.stdenv.hostPlatform;
            }
          );
          buildGoModule = pkgs.buildGoModule.override { inherit stdenv; };
        in
        {
          default = (
            (pkgs.callPackage ./pkg.nix (params // { inherit buildGoModule; })).overrideAttrs (old: {
              env = stdenv.hostPlatform.go // {
                CGO_ENABLED = "0";
              };
              doCheck = false;
            })
          );
        }
      );
  };
}
