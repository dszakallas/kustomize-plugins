{
  pkgs,
  lib,
  config,
  ...
}:

{
  languages.go.enable = true;

  packages = [ pkgs.git ];

  enterShell = ''
    go version
  '';

  enterTest = ''
    make test
  '';

}
