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

      # nix develop: Go and Python for go test / the release scripts, g++ and Fcitx5 for platforms/linux-fcitx5,
      # Rust and WebKitGTK for the settings app (platforms/settings), and NumPy + ONNX Runtime for the neural workers.
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              gotools
              (python3.withPackages (ps: [
                ps.numpy
                ps.onnxruntime
              ]))
              gnumake
              pkg-config
              jq
              xz
              fcitx5
              cargo
              rustc
              webkitgtk_4_1
              gtk3
              libsoup_3
              glib
              glib-networking
            ];
            # go.mod pins the language version; never let go download another toolchain.
            GOTOOLCHAIN = "local";
          };
        }
      );

      # Builds with the system's nixpkgs, so lib0c.so links the same libFcitx5Core as the running Fcitx5.
      overlays.default = final: prev: {
        zc-ime = final.callPackage ./nix/package.nix { };
      };
    };
}
