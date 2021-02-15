let
  pkgs = import <nixpkgs> {};
  kubebuilder = pkgs.callPackage .nix/kubebuilder.nix {};
in
pkgs.mkShell {
    buildInputs = with pkgs; [
      go
      kubebuilder
      kind
      kubectl
      kustomize
    ];
}
