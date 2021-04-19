let
  pkgs = import <nixpkgs> {};
  unstable = import (fetchTarball https://github.com/NixOS/nixpkgs-channels/archive/nixos-unstable.tar.gz) { };
  kubebuilder = pkgs.callPackage .nix/kubebuilder.nix {};
  operator-sdk = pkgs.callPackage .nix/operator-sdk.nix {};
in
pkgs.mkShell {
    buildInputs = with pkgs; [
      go
      kubebuilder
      operator-sdk
#      unstable.kind
      kubectl
      kustomize
    ];
}
