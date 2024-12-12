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
  inherit
    vendorHash
    version;
  pname = "incus";
  src = ./..;

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
