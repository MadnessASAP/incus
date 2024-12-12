{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-24.11";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = { self, flake-parts, ... }@inputs:
    let
      rev = self.shortRev or self.dirtyShortRev;
      version = "6.7-${rev}";
      vendorHash = "sha256-u12zYcKiHNUH1kWpkMIyixtK9t+G4N2QerzOGsujjFQ=";
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      perSystem = { self', pkgs, ... }: {
        packages = rec {
          default = incus;
          incus = pkgs.callPackage ./flake/incus.nix { inherit version vendorHash; };
          client = pkgs.callPackage ./flake/client.nix { inherit version vendorHash; };
        };

        checks = {
          staticAnalysis = self'.packages.incus.overrideAttrs (prev: {
            pname = "incus-static-analysis";
            nativeCheckInputs = with pkgs; [
              debianutils
              gettext
              git
              go-licenses
              golangci-lint
              shellcheck
              (python3.withPackages (pyPkgs: with pyPkgs; [
                flake8
              ]))
              (pkgs.buildGoModule rec {
                pname = "xgettext-go";
                version = "2.57.1";
                vendorHash = "sha256-e1QFZIleBVyNB0iPecfrPOg829EYD7d3KMHIrOYnA74=";
                src = pkgs.fetchFromGitHub {
                  owner = "canonical";
                  repo = "snapd";
                  rev = version;
                  hash = "sha256-icPEvK8jHuJO38q1n4sabWvdgt9tB5b5Lh5/QYjRBBQ=";
                };
                subPackages = [ "i18n/xgettext-go" ];
              })
            ];
            INCUS_OFFLINE = 1;
            dontBuild = true;
            checkPhase = "make static-analysis";
          });
        };

        devShells.default = pkgs.callPackage ./flake/shell.nix { };
        formatter = pkgs.nixpkgs-fmt;
      };
    };
}
