{
  vendorHash,
  version,

  buildGoModule,
}: buildGoModule {
  inherit
    vendorHash
    version;
  pname = "incus-client";
  src = ./..;

  subPackages = [ "cmd/incus" ];
}
