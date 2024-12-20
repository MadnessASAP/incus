{
  description = "Incus is a modern, secure and powerful system container and virtual machine manager.";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-parts.url = "flake-parts";
  };

  outputs = { flake-parts, ... }@inputs:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
      ];
      perSystem = { pkgs, ... }: {
        devShells.default = pkgs.callPackage ./.flake/shell.nix { };
        formatter = pkgs.nixpkgs-fmt;
      };
    };
}
