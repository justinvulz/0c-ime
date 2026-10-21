# NixOS

repo 根目錄的 `flake.nix` 把靈犀注音建成 Fcitx5 addon：外殼 `lib0c.so`、引擎、`0c-update`、`0c-mcp`、設定程式與字典，以 [Linux 系統安裝](../platforms/linux-system/README.md) 的 `make install` 裝進 Nix store，由 Nix 管理更新。

| flake 輸出 | 說明 |
|---|---|
| `overlays.default` | 加入 `pkgs.zc-ime`，以系統的 nixpkgs 建置（建議） |
| `packages.<system>.zc-ime`（`default`） | 以 flake 自己的 nixpkgs 建置 |
| `packages.<system>.zc-ime-with-models` | 同上，內附已配對的神經模型 |

請用 overlay：Fcitx5 外殼是載入 Fcitx5 行程的共用函式庫，要以系統同一份 nixpkgs 建置，`lib0c.so` 才會連到與執行中 Fcitx5 相同的 `libFcitx5Core`。

## 加到系統設定

需要開啟 flakes：`nix.settings.experimental-features = [ "nix-command" "flakes" ];`

### 系統設定是 flake

```nix
# /etc/nixos/flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    zc-ime = {
      url = "github:0c-biggo/0c-ime";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { nixpkgs, zc-ime, ... }: {
    nixosConfigurations.<主機名稱> = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        { nixpkgs.overlays = [ zc-ime.overlays.default ]; }
        ./configuration.nix
      ];
    };
  };
}
```

```nix
# /etc/nixos/configuration.nix
{ pkgs, ... }:
{
  i18n.inputMethod = {
    enable = true;
    type = "fcitx5";
    fcitx5.addons = [ pkgs.zc-ime ];
  };
  environment.systemPackages = [ pkgs.zc-ime ]; # 設定程式、應用程式選單項目與圖示
}
```

### 系統設定是傳統的 configuration.nix

以 `builtins.getFlake` 取用，其餘同上：

```nix
# /etc/nixos/configuration.nix
{ pkgs, ... }:
let
  zc-ime = builtins.getFlake "github:0c-biggo/0c-ime/<commit>";
in
{
  nixpkgs.overlays = [ zc-ime.overlays.default ];

  i18n.inputMethod = {
    enable = true;
    type = "fcitx5";
    fcitx5.addons = [ pkgs.zc-ime ];
  };
  environment.systemPackages = [ pkgs.zc-ime ];
}
```

`<commit>` 固定版本；不帶時每次重建可能取到不同的 commit。用本機的 clone 時寫 `git+file:///path/to/0c-ime`（只讀 Git 追蹤的檔案）。

## 加入輸入法

```sh
sudo nixos-rebuild switch
```

重新登入（或 `fcitx5 -r`），在 Fcitx5 設定（`fcitx5-configtool`）把「靈犀」加入輸入法清單。

同一個帳號另有官網安裝（`~/.local/bin/0c-server`）時，Fcitx5 先讀家目錄的 addon 設定，用的是官網那一份；只想用 Nix 的版本請先移除官網安裝。

## 選項

以 `override` 調整：

```nix
{ pkgs, ... }:
let
  zc-ime = pkgs.zc-ime.override { withModels = true; };
in
{
  i18n.inputMethod.fcitx5.addons = [ zc-ime ];
  environment.systemPackages = [ zc-ime ];
}
```

| 選項 | 預設 | 說明 |
|---|---|---|
| `withModels` | `false` | 內附已配對的神經模型（約 250 MB，取自 Release 的模型 tarball），不必在設定程式下載，也不提醒下載 |
| `withNeural` | `true` | 附 Python、NumPy、ONNX Runtime 給神經選字與補標點；`false` 時只用統計選字，也不提供下載模型 |
| `withSettings` | `true` | 設定程式（`0c-settings`） |
| `withIbus` | `false` | 一併建 IBus 外殼；改用 IBus 時放進 `i18n.inputMethod.ibus.engines` |

## 神經模型

預設不內附模型。在設定程式的「模型」頁按「下載模型」，從 Hugging Face 下載到 `~/.local/share/zuin/models/`，不會在背景自動下載；推論用的 Python 環境已經包在套件裡，不必另外安裝。沒有模型時以統計引擎選字。

## 更新

這份安裝由 Nix 管理（`share/0c/distribution.json` 的 `updates` 是 `package-manager`）：設定程式的「更新」頁只顯示目前版本，`0c-update` 不會檢查或下載更新。要更新時：

```sh
nix flake update zc-ime        # 系統設定是 flake；getFlake 則改 <commit>
sudo nixos-rebuild switch
```

執行中的 Fcitx5 要重新啟動才會載入新版外殼。

個人資料在 `~/.local/share/zuin`（`user.json`、詞彙、下載的模型）與 `~/.local/share/0c/settings.json`，更新或移除套件都不會動到。

## 只建置不安裝

```sh
nix build github:0c-biggo/0c-ime   # 結果連到 ./result；--no-link 不建連結
```

## 換版本（維護者）

`nix/package.nix` 的 `version` 與三個雜湊要一起更新；建置時會核對 Release 的字典與這份原始碼的 `data/dictionary.lock.json` 相符。

```sh
nix store prefetch-file https://github.com/0c-biggo/0c-ime/releases/download/v<版本>/0c-ime-<版本>.tar.xz          # sourceTarball
nix store prefetch-file https://github.com/0c-biggo/0c-ime/releases/download/v<版本>/0c-ime-models-<版本>.tar.xz   # modelsTarball
```

`go.mod`／`go.sum` 改變時，`vendorHash` 改成 `lib.fakeHash` 建置一次，換成錯誤訊息 `got:` 的值。`platforms/settings/Cargo.lock` 直接讀取，不需要雜湊。
