let
  pkgs = import <nixpkgs> { };
in
pkgs.mkShell {
  packages = with pkgs; [
    # dev environment
    go
    golangci-lint
    gopls

    # static-analysis
    debianutils
    shellcheck
    (python3.withPackages (pyPkgs: with pyPkgs; [
      flake8
    ]))
  ];
  inputsFrom = [
    pkgs.incus
  ];
}
