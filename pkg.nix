{
  buildGoModule,
  fetchFromGitHub,
  stdenv,
  lib,
  vendorHash,
  version,
}:

buildGoModule rec {
  pname = "kustomize-plugins";
  inherit vendorHash version;

  src = ./.;

  env.CGO_ENABLED = "0";

  ldflags = [
    "-s"
    "-w"
  ];

  subPackages = [ 
    "cmd/resourceinjector"
    "cmd/yqtransform"
  ];

  checkPhase = ''
    make test
  '';

  postInstall = ''
    if [ -d "$out/bin" ]; then
      for f in "$out/bin"/*; do
        [ -e "$f" ] || continue
        base="$(basename "$f")"
        mv "$f" "$out/bin/kustomize-plugin-$base"
      done
    fi
  '';

  meta = {
    license = lib.licenses.mit;
  };
}
