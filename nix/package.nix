# 靈犀注音 0C as a Fcitx5 addon: the shell (lib0c.so), the engine, 0c-update, 0c-mcp, the settings app and the
# dictionary, installed with the root Makefile's system install (platforms/linux-system/README.md) into $out.
# distribution.json marks it package-managed, so 0c-update never checks for or downloads updates.
{
  lib,
  stdenv,
  buildGoModule,
  fetchurl,
  go,
  pkg-config,
  jq,
  cargo,
  rustc,
  rustPlatform,
  wrapGAppsHook3,
  fcitx5,
  webkitgtk_4_1,
  gtk3,
  libsoup_3,
  glib,
  glib-networking,
  python3,
  # Settings app (Tauri 2).
  withSettings ? true,
  # IBus shell; NixOS's ibus-with-plugins reads share/ibus/component.
  withIbus ? false,
  # Python with NumPy and ONNX Runtime for the neural selection and punctuation workers. Without it the engine
  # uses the statistical ranking only.
  withNeural ? true,
  # Ship the paired models from the release's models tarball (~250 MB) instead of letting the settings app
  # download them from Hugging Face into ~/.local/share/zuin/models.
  withModels ? false,
}:

assert withModels -> withNeural;

let
  # Bump together with the two release tarball hashes below. The dictionary must match data/dictionary.lock.json
  # of this tree (checked in postPatch).
  version = "0.2.6";

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.difference ../. (
      lib.fileset.unions [
        ./.
        ../flake.nix
        (lib.fileset.maybeMissing ../flake.lock)
        (lib.fileset.maybeMissing ../assets)
        (lib.fileset.maybeMissing ../build)
        ../platforms/macos
        ../platforms/windows
      ]
    );
  };

  release = "https://github.com/0c-biggo/0c-ime/releases/download/v${version}";

  # The compiled dictionary (assets/data) is not in git; it ships in the release source tarball.
  sourceTarball = fetchurl {
    url = "${release}/0c-ime-${version}.tar.xz";
    hash = "sha256-7NzG9kQEnU+JZFXoTb3hzmLRJXDEVr0b7TnS9JH9qFo=";
  };

  modelsTarball = fetchurl {
    url = "${release}/0c-ime-models-${version}.tar.xz";
    hash = "sha256-YgreNuCB12iiJeVkJje6xUSlmvnsMklMJGjQGavlI38=";
  };

  # Only used for its vendored Go modules; the binaries are built by the Makefile below.
  goModules =
    (buildGoModule {
      pname = "0c-ime-go-modules";
      inherit version src;
      vendorHash = "sha256-FfxcQMKparYVmx0JMnsybSRB78/8iqRVL8zcfcBs+6w=";
    }).goModules;

  pythonEnv = python3.withPackages (ps: [
    ps.numpy
    ps.onnxruntime
  ]);

  flag = b: if b then "1" else "0";
in
stdenv.mkDerivation (
  {
    pname = "0c-ime";
    inherit version src;

    strictDeps = true;

    nativeBuildInputs = [
      go
      pkg-config
      jq
    ]
    ++ lib.optionals withSettings [
      cargo
      rustc
      rustPlatform.cargoSetupHook
      wrapGAppsHook3
    ];

    buildInputs = [
      fcitx5
    ]
    ++ lib.optionals withSettings [
      webkitgtk_4_1
      gtk3
      libsoup_3
      glib
      glib-networking
    ];

    postPatch = ''
      mkdir -p "$TMPDIR/release" assets
      tar -xJf ${sourceTarball} -C "$TMPDIR/release" --wildcards '*/assets/data/*'
      cp -r "$TMPDIR"/release/*/assets/data assets/data
      jq -r '.files | to_entries[] | "\(.value.sha256)  assets/data/\(.key)"' data/dictionary.lock.json \
        | sha256sum -c --quiet -
    ''
    + lib.optionalString withNeural ''
      substituteInPlace internal/neural/bundle.go \
        --replace-fail '"/usr/bin/python3"' '"${pythonEnv}/bin/python3"'
    '';

    preBuild = ''
      export GOCACHE="$TMPDIR/go-cache" GOPATH="$TMPDIR/go" GOFLAGS=-mod=vendor GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
      cp -r --reflink=auto ${goModules} vendor
      chmod -R u+w vendor
    '';

    makeFlags = [
      "PREFIX=${placeholder "out"}"
      "VERSION=${version}"
      "GO_MODULE=github.com/0c-biggo/0c-ime"
      "IBUS_COMPONENTDIR=${placeholder "out"}/share/ibus/component"
      "FCITX5=1"
      "IBUS=${flag withIbus}"
      "SETTINGS=${flag withSettings}"
      "MODEL_DOWNLOAD=${flag (withNeural && !withModels)}"
    ];

    buildFlags = [ "build" ];

    postInstall = lib.optionalString withModels ''
      mkdir -p "$TMPDIR/models"
      tar -xJf ${modelsTarball} -C "$TMPDIR/models"
      models=$(dirname "$(find "$TMPDIR/models" -type d -path '*/models/select' -print -quit)")
      cp -r --no-preserve=mode "$models/select" "$models/punct" $out/share/0c/neural/
    '';

    # Only the settings app gets the GTK wrapper: 0c-update and 0c-server check their own path against
    # libexec/0c, which a wrapper script would break.
    dontWrapGApps = true;
    postFixup = lib.optionalString withSettings ''
      wrapGApp $out/bin/0c-settings
    '';

    meta = {
      description = "Zhuyin (Bopomofo) input method for Fcitx5 that picks homophones with an on-device Transformer";
      homepage = "https://0c.biggo.com";
      license = lib.licenses.agpl3Only;
      platforms = lib.platforms.linux;
      mainProgram = "0c-settings";
      isIbusEngine = withIbus;
    };
  }
  // lib.optionalAttrs withSettings {
    cargoRoot = "platforms/settings";
    cargoDeps = rustPlatform.importCargoLock { lockFile = ../platforms/settings/Cargo.lock; };
  }
)
