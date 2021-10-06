let
  pkgs = import <nixpkgs> {};
  operator-sdk = pkgs.callPackage .nix/operator-sdk.nix {};
in
pkgs.mkShell {
    buildInputs = with pkgs; [
      go
      operator-sdk
      kind
      kubectl
      kustomize
    ];
}
