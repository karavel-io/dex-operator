{ stdenv, fetchurl, pkgs, lib }:
let
  name = "kubebuilder";
  version = "2.3.1";
  sys = lib.systems.parse.mkSystemFromString builtins.currentSystem;
  os = sys.kernel.name;
  arch = if lib.systems.inspect.predicates.isx86_64 sys then "amd64"
         else sys.cpu.arch;
in
stdenv.mkDerivation {
  name = "${name}";

  src = fetchurl {
    url = "https://go.kubebuilder.io/dl/${version}/${os}/${arch}";
    sha256 = "047dljgba5q1g6sp5fdkhl0d1s1dzk7bv8mby9infw09y9q6jjgz";
  };

  unpackPhase = ''
    tar -xzf $src
  '';
  installPhase = ''
    mkdir -p $out/bin
    cp ${name}_${version}_${os}_${arch}/bin/${name} $out/bin/${name}
  '';

  meta = {
    description = "SDK for building Kubernetes APIs using CRDs";
    longDescription = ''
      Kubebuilder is a framework for building Kubernetes APIs using custom resource definitions (CRDs).
      Similar to web development frameworks such as Ruby on Rails and SpringBoot, Kubebuilder increases velocity and reduces the complexity managed by developers for
      rapidly building and publishing Kubernetes APIs in Go. It builds on top of the canonical techniques used to build the core Kubernetes APIs to provide simple abstractions
      that reduce boilerplate and toil.
    '';
    homepage = "https://kubebuilder.io/";
    license = "Apache-2.0";
    maintainers = [
      stdenv.lib.maintainers.matteojoliveau
    ];
  };
}
