{
  mkShell,

  go,
  golangci-lint,
  gopls,
}: mkShell {
  packages = [
    go
    golangci-lint
    gopls
  ];
}
