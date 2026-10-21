{
  description = "靈犀注音 0C — Zhuyin input method as a Fcitx5 addon";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
      ];
    in
    {
      packages = forAllSystems (
        system:
        let
          zc-ime = nixpkgs.legacyPackages.${system}.callPackage ./nix/package.nix { };
        in
        {
          inherit zc-ime;
          zc-ime-with-models = zc-ime.override { withModels = true; };
          default = zc-ime;
        }
      );

      # Builds with the system's nixpkgs, so lib0c.so links the same libFcitx5Core as the running Fcitx5.
      overlays.default = final: prev: {
        zc-ime = final.callPackage ./nix/package.nix { };
      };
    };
}
