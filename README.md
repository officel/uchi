# uchi

Dotfiles Literate Configuration CLI tool.

## uchi とは

- リテレートコンフィギュレーション（Literate Configuration）の思想に基づき、dotfiles などのツール設定を Markdown ファイルで一元管理します。
- `uchi: v1` フォーマットで記述された Markdown 内のコードフェンスから、`alias`, `env`, `profile`, `rc`, `function` などのスキーマごとに整理・抽出された設定ファイルを生成します。
- 既存の dotfiles 管理や chezmoi 等のツールと競合しないよう、Markdown ファイルの解析と設定ファイルの出力（生成）のみを行います。
- 入力ディレクトリ（既定値: `.`）配下の Markdown を処理し、出力ディレクトリ（既定値: `../dist/`）へ書き出します。
  - 分割ファイル: `../dist/parts/<relative_path>/<schema>`
  - 統合ファイル: `../dist/<schema>`
- 抽出対象のコードブロックは `-s` / `--shell` オプションおよびコードフェンスの `target` 属性によって選別されます。
- 入力文書フォーマットの全仕様については [docs/format-v1.md](docs/format-v1.md) を参照してください。

## クイックスタート手順（導入例）

初めて利用する際の一連の手順例です。初期化から設定ファイルの作成、検証、生成、差分確認まで順を追って体験できます。

### 1. ビルド

リポジトリ直下で CLI バイナリをビルドします。

```bash
go build -o uchi ./cmd/uchi
```

### 2. 設定ファイルの初期化 (`init`)

カレントディレクトリに既定の設定ファイル `.uchi.yaml` を生成します。

```bash
./uchi init
```

*出力例:*

```text
Created ./.uchi.yaml
```

### 3. Markdown ドキュメントの作成 (`new`)

テンプレートから設定管理用の Markdown ファイルを作成します。

```bash
./uchi new shell
```

*出力例:*

```text
Created shell.md
```

生成された `shell.md` には `uchi: v1` の frontmatter と、各種 schema (`alias`, `env` 等) のコードフェンスが含まれています。

### 4. 入力文書と設定の検証 (`check`)

作成した Markdown の frontmatter やコードフェンス属性に誤りがないか検証します。

```bash
./uchi check
```

*出力例:*

```text
Config file: found (./.uchi.yaml)
Options:
  input_dir: .
  output_dir: ../dist
  template_dir:
  auto_comment: true
```

> **目的 (検証)**: ディスクへの書き込みを行わず、Markdown 構文・アノテーション・シェル互換性のエラーを事前検出します。

### 5. 生成計画の事前確認 (`gen --dry-run`)

実際の生成処理を行わずに、出力対象となるファイルパス一覧を確認します。

```bash
./uchi gen --dry-run
```

*出力例:*

```text
../dist/parts/shell/alias
../dist/parts/shell/env
../dist/parts/shell/profile
../dist/parts/shell/rc
../dist/parts/shell/function
../dist/alias
../dist/env
../dist/profile
../dist/rc
../dist/function
```

> **目的 (事前確認・プレビュー)**: ファイルシステムを変更せず、計画されている生成ターゲットパスを検証します。

### 6. 設定ファイルの生成 (`gen`)

抽出処理を実行し、出力ディレクトリ（例: `./dist` を指定）へ設定ファイルをアトミックに書き出します。

```bash
./uchi gen -o ./dist
```

> **目的 (生成)**: 実際の分割・統合設定ファイルおよびマニフェストファイル (`.uchi-manifest.json`) を生成・保存します。

### 7. 差分確認 (`diff`)

生成計画および既存ファイル・マニフェストとの差分を確認します。

```bash
./uchi diff -o ./dist
```

*出力例 (変更なしの場合):*

```text
[=] dist/parts/shell/alias (kind: part, status: unchanged)

[=] dist/parts/shell/env (kind: part, status: unchanged)
...
```

`shell.md` を編集して新しい `alias` を追加した後に再度 `./uchi diff -o ./dist` を実行すると、以下のように変更内容が表示されます。

```text
[~] dist/parts/shell/alias (kind: part, status: modified)
  # shell
  alias g="git"
+ alias ll="ls -la"
```

> **目的 (差分確認)**: ファイルを上書き・適用する前に、どのような追加 (`[+]`)、変更 (`[~]`)、削除 (`[-]`) が生じるかを正確に確認します。

---

## 対象シェル指定と自動変換に関する制限

- **既定の対象シェル**: `-s` / `--shell` オプション未指定時の既定値は `all` です。
- **対応するシェル種別**: `all`, `bash`, `fish`, `powershell`, `pwsh`, `sh`, `zsh`
- **構文変換の範囲**:
  - `bash` および `zsh` を対象とする場合、一部の安全かつ明確な構文変換（例: `env` スキーマにおける Zsh `typeset -x` から Bash `export` への自動変換）をサポートしています。
  - 対象シェルで非互換または未対応の構文が含まれる場合は、推測して変換せずエラー診断メッセージが出力されます。
  - `PowerShell` やその他のシェルに対する自動的な構文相互変換機能は現時点で未実装です。

---

## 出力ディレクトリのバージョン管理に関する注意点 (`.gitignore`)

`uchi gen` によって出力されるディレクトリ（デフォルト: `../dist/` や指定した `-o` パス）および生成マニフェスト (`.uchi-manifest.json`) は、Markdown ソースから再現可能な生成物（アーティファクト）です。

意図しない生成物のリポジトリ混入を防ぐため、出力ディレクトリを `.gitignore` に追加しておくことを推奨します。

```gitignore
# uchi 出力ディレクトリの除外例
/dist/
../dist/
.uchi-manifest.json
```

---

## 詳細ドキュメント

- [BUILD.md](BUILD.md): 全サブコマンド、フラグ仕様、終了コード、詳細な実行例
- [docs/format-v1.md](docs/format-v1.md): `uchi: v1` 入力 Markdown フォーマット仕様書
