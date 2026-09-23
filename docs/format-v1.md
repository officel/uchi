# uchi 入力 Markdown フォーマット仕様 (v1)

本ドキュメントは、`uchi`（Literate Configuration CLI）が処理する入力 Markdown ファイルの構造、属性、出力ルール、およびエラー／無視の扱いを明文化した仕様書です。

## 概要

`uchi` は Markdown ファイルで記述された設定ドキュメントから、指定された schema に基づいて各種ツールの設定コードブロックを抽出し、分割・統合された設定ファイルを生成します。

フォーマットバージョン `v1` において、入力文書は **frontmatter** と **schema 属性を持つコードフェンス** によって構成されます。

---

## 適用対象ファイル

`uchi` の抽出処理（`gen` サブコマンド）において、処理対象となるファイルは以下の条件を満たすものです。

- 入力ディレクトリ（`-i` / `--input`、既定値: `.`）配下のディレクトリを再帰的に走査して検出されるファイル。
- 拡張子が `.md` または `.markdown` であるファイル。
- その他の拡張子を持つファイルおよびディレクトリ自体は処理対象外としてスキップされます。

---

## Frontmatter 仕様

処理対象の Markdown ファイルの先頭には、YAML 形式の frontmatter を記述します。

### 必須要素

- ファイルの 1 行目が `---` で始まり、それ以降の行に存在する `---` までの間が frontmatter として解析されます。
- frontmatter 内に **`uchi: v1`** という文字列が含まれている必要があります。

```markdown
---
uchi: v1
---
```

### 任意要素

- `title`, `description`, `date` などの任意のメタデータキーを記述できます。
- `uchi` は `uchi: v1` の文字列の存在チェックのみを行い、その他の frontmatter 内容は抽出処理に影響を与えず保持・無視されます。

```markdown
---
uchi: v1
title: Bash Configuration
description: Bash aliases and environment variables
---
```

### 無視される条件

- frontmatter 自体が存在しない場合、または frontmatter 内に `uchi: v1` が含まれていない場合、そのファイルはエラーにならず**無視（処理をスキップ）**されます。

---

## コードフェンスと schema 属性

抽出対象となるコードブロックは、バックティック 3 つ（` ``` `）で囲まれたコードフェンス形式で記述します。

### アノテーション構文

コードフェンスの開始行（ヘッダー）は以下の形式でアノテーションを指定します。

```text
```[language] {[attributes]}
```

- **`language`**（任意）:
  - ブロックの言語識別子（例: `bash`, `zsh`, `sh` など）。アノテーションの中括号 `{` の前に記述します。
- **アノテーションブロック `{ ... }`**:
  - 波かっこ `{` と `}` で囲まれた属性定義。
  - スペース区切りで `key=value` または単体キー `key`（`key="true"` として解釈）を指定します。
- **`schema` 属性**:
  - アノテーション内に `schema=<schemaName>` の形式で抽出先の schema 名を指定します。

### コードブロック抽出の成立条件

以下の条件をすべて満たすコードブロックのみが抽出処理の対象となります。

1. 開始フェンスに波かっこ `{ ... }` によるアノテーションが存在すること。
2. アノテーション内に `schema` 属性（`schema=<schemaName>`）が指定されていること。
3. 指定された `schemaName` が「既知の Schema」に該当すること。

---

## 既知 Schema

現行の `v1` 仕様において定義されている既知の schema 名は以下の 5 種類です。

| Schema 名 | 用途 |
|---|---|
| `alias` | コマンドエイリアス定義 |
| `env` | 環境変数定義 |
| `profile` | プロファイル・ログインシェル設定 |
| `rc` | 実行時設定（rc ファイル） |
| `function` | シェル関数定義 |

### 未定義 Schema の扱い（現状と将来の変更予定）

- **現状の挙動**:
  - 上記 5 種類以外の未定義 schema（例: `{schema=custom}`）が指定された場合、現状の実装ではエラーとせず、静かに**無視（抽出をスキップ）**します。
- **将来の変更予定**:
  - 未定義 schema の扱いは現状の動作（無視）として明記しますが、後続の仕様変更において、エラーとするかカスタム schema を許容する等の仕様変更が予定されています。

---

## 出力構造と出力ルール

`uchi gen` コマンドを実行した際、抽出結果は指定された出力ディレクトリに以下の構造で生成されます。

### 既定値 (Default Options)

- **`input_dir`**: `.`（カレントディレクトリ）
- **`output_dir`**: `../dist`
- **`auto_comment`**: `true`（自動ヘッダーコメント付与機能が有効）

### 1. 分割出力 (`parts/` ディレクトリ)

入力ファイルごと・schema ごとに抽出されたコードブロックを出力します。

- **出力パス**: `<output_dir>/parts/<relative_path_without_ext>/<schema>`
  - 例: 入力 `bash.md` （`schema=alias`） $\rightarrow$ `<output_dir>/parts/bash/alias`
  - 例: 入力 `tools/git.markdown` （`schema=env`） $\rightarrow$ `<output_dir>/parts/tools/git/env`
- **結合ルール**:
  - 同一ファイル内に同じ `schema` を持つコードブロックが複数存在する場合、空でないブロックの内容が改行（`\n`）区切りで結合されます。
- **自動ヘッダーコメント (`auto_comment`)**:
  - `auto_comment` オプションが `true`（既定値）かつ抽出コンテンツが存在する場合、出力ファイルの先頭に `# <tool_name>\n`（`<tool_name>` は拡張子を除いた相対パス、パス区切りは `/`）が自動付与されます。

### 2. 統合出力 (出力ディレクトリ直下)

抽出対象となったすべての Markdown ファイルから、schema ごとに集約した統合ファイルを出力します。

- **出力パス**: `<output_dir>/<schema>`
  - 例: `<output_dir>/alias`, `<output_dir>/env`
- **結合ルール**:
  - 各ファイルで抽出・生成された分割コンテンツ（ヘッダーコメントを含む）が、空行（`\n\n`）区切りで結合されます。

### 3. 改行・空コンテンツのフォーマット

- 生成されるすべての出力ファイルは、末尾が単一の改行（`\n`）で終わります。
- 抽出結果が空または空白のみの場合は、単一の改行（`\n`）のみが書き出されます。

---

## 具体例

新規利用者が 1 つの Markdown ファイルから `gen` コマンドの出力を再現できる具体例です。

### 1. 正常例 (Valid Example)

#### 入力 Markdown ファイル (`input/shell.md`)

````markdown
---
uchi: v1
title: Shell Configuration
---

# Shell Config

```bash {schema=alias}
alias ll='ls -la'
````

ここにメモ書きやドキュメントテキストが入ります。この部分は抽出されません。

```bash {schema=env}
export EDITOR=vim
```

```bash {schema=alias}
alias gs='git status'
```

````text

#### `uchi gen -i input -o dist` 実行後の生成ファイル構造

```text
dist/
├── alias
├── env
└── parts/
    └── shell/
        ├── alias
        └── env
```

#### 各出力ファイルの内容

- **`dist/parts/shell/alias`**:

  ```text
  # shell
  alias ll='ls -la'
  alias gs='git status'
  ```

- **`dist/parts/shell/env`**:

  ```text
  # shell
  export EDITOR=vim
  ```

- **`dist/alias`**:

  ```text
  # shell
  alias ll='ls -la'
  alias gs='git status'
  ```

- **`dist/env`**:

  ```text
  # shell
  export EDITOR=vim
  ````

---

### 2. 無視される例 (Ignored Inputs)

以下の入力はエラーを発生させず、静かに無視（抽出処理からスキップ）されます。

#### Case A: Frontmatter が無いファイル

````markdown
# Title Only

```bash {schema=alias}
alias ll='ls -la'
```
````

- **理由**: 先頭に `---` の frontmatter が無いためファイル全体がスキップされます。

#### Case B: frontmatter 内に `uchi: v1` が無いファイル

````markdown
---
title: General Note
---

```bash {schema=alias}
alias ll='ls -la'
```
````

- **理由**: `uchi: v1` が含まれていないためスキップされます。

#### Case C: アノテーション `{}` が無いコードブロック

````markdown
---
uchi: v1
---

```bash
alias ll='ls -la'
```
````

- **理由**: `{ ... }` によるアノテーションが無いため抽出対象外です。

#### Case D: `schema` 属性が無いコードブロック

````markdown
---
uchi: v1
---

```bash {lang=bash}
alias ll='ls -la'
```
````

- **理由**: アノテーション内に `schema` 属性が無いためスキップされます。

#### Case E: 未定義の Schema を指定したコードブロック (現状)

````markdown
---
uchi: v1
---

```bash {schema=custom}
echo "hello"
```
````

- **理由**: `custom` は既知の Schema (`alias`, `env`, `profile`, `rc`, `function`) に含まれないため、現状の実装ではスキップされます。

---

### 3. エラーとなる入力 (Error Inputs)

- **ファイルシステム権限エラー**:
  - 入力ディレクトリが読み取り不可、または出力ディレクトリ・ファイルの作成・書き込みが権限不足等で失敗した場合。
- **無効な CLI フラグ / 引数指定**:
  - 不正なサブコマンドや存在しないフラグを指定した場合。
- **注（Markdown 構文の扱い）**:
  - 閉じられていない `---` や閉じられていない ` ``` ` などの不完全な Markdown 構文であっても、スキャナがファイル末尾（EOF）までを 1 つのブロックとして読み込むため、解析エラー（`error`）は発生せず、読み込まれた範囲の文字列として処理されます。

---

## 後方互換性と将来のバージョン追加時の扱い

### 後方互換性 (Backward Compatibility)

- `uchi: v1` を指定した入力ドキュメントは、今後の CLI バージョンアップにおいても互換性が維持され、本仕様に従って処理されます。

### 将来のバージョン追加時の扱い (Future Versions)

- 将来 `v2` などの新しい仕様バージョンが追加された場合、frontmatter 内の `uchi: <version>` 指定（例: `uchi: v2`）によって処理ロジックが切り替えられます。
- `v1` 解析器は frontmatter 内の `uchi: v1` の存在で認識するため、`uchi: v2` のみが指定されたドキュメントは `v1` 解析器からは無視（スキップ）されます。
- これにより、上位バージョンで非互換な新記法や機能拡張が追加された場合でも、既存の `v1` ドキュメントの動作に影響を与えません。
