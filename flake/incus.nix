{ lib
, # From flake
  vendorHash
, version
, # Builders
  buildGoModule
, pkg-config
, # Dependencies
  acl
, cowsql
, libcap
, lxc
, sqlite
, udev
}:
let
  inherit (lib)
    concatStringsSep
    map
    pipe;
in
buildGoModule {
  pname = "incus";
  src = ./..;
  inherit
    vendorHash
    version;

  nativeBuildInputs = [
    pkg-config
  ];

  buildInputs = [
    lxc
    acl
    libcap
    cowsql.dev
    sqlite
    udev.dev
  ];

  checkFlags =
    let
      skippedTests = pipe [
        "TestContainerTestSuite"
        "TestConvertNetworkConfig"
        "TestConvertStorageConfig"
        "TestSnapshotCommon"
        "TestValidateConfig"
      ] [
        (map (test: "^${test}$"))
        (concatStringsSep "|")
      ];
    in
    [ "-skip=${skippedTests}" ];
}
