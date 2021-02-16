{ lib, buildGoModule, fetchFromGitHub }:

buildGoModule rec {
    pname = "kubebuilder";
    version = "2.3.1";

    src = fetchFromGitHub {
        owner = "kubernetes-sigs";
        repo = pname;
        rev = "v${version}";
        sha256 = "07frc9kl6rlrz2hjm72z9i12inn22jqykzzhlhf9mcr9fv21s3gk";
    };

    vendorSha256 = "079cnaflk2ap5fb357151fdqk7wr37dpghd3lmrmhcqwpfwp022m";

    subPackages = [ "cmd" ];

    postInstall = ''
    mv $out/bin/cmd $out/bin/${pname}
    '';

    meta = with lib; {
        description = "SDK for building Kubernetes APIs using CRDs";
        longDescription = ''
          Kubebuilder is a framework for building Kubernetes APIs using custom resource definitions (CRDs).
          Similar to web development frameworks such as Ruby on Rails and SpringBoot, Kubebuilder increases velocity and reduces the complexity managed by developers for
          rapidly building and publishing Kubernetes APIs in Go. It builds on top of the canonical techniques used to build the core Kubernetes APIs to provide simple abstractions
          that reduce boilerplate and toil.
        '';
        homepage = "https://kubebuilder.io/";
        license = licenses.asl20;
        maintainers = [
          stdenv.lib.maintainers.matteojoliveau
        ];
      };
}

