{
  inputs = {
    nixpkgs.url = "nixpkgs";
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
      ];

      perSystem = { pkgs, ... }: {
        packages.incus = pkgs.callPackage ./flake/incus.nix { inherit version vendorHash; };

        devShells = {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              golangci-lint
              gopls
            ];
          };
        };

        formatter = pkgs.nixpkgs-fmt;
      };
    };
}
