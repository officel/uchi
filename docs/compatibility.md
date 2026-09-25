# uchi 互換性方針および性能計測手順

本ドキュメントは、`uchi`（Literate Configuration CLI）における互換性保証方針、破壊的変更の扱い、ゴールデンテストによる回帰防止機構、および性能基準と計測手順について定義します。

---

## 1. 互換性方針 (Compatibility Policy)

`uchi` は教育的かつ小規模な CLI ツールとして、シンプルで予測可能な動作を提供することを最優先とします。

### v1 フォーマットの互換性保証

- **`uchi: v1` 仕様の維持**:
  - frontmatter に `uchi: v1` を宣言している入力 Markdown 文書は、今後のバージョンアップデートにおいても動作および出力構造が完全に保護されます。
  - `uchi: v1` フォーマットの全仕様については [docs/format-v1.md](format-v1.md) に準拠します。
- **既存出力の意図しない変更防止**:
  - 同一の `uchi: v1` 入力からは、同じ `uchi` バージョンおよび設定オプションのもとで常にバイト単位で同一の出力ファイル（分割・統合出力）が生成されます。

### 破壊的変更と移行方針 (Breaking Changes & Migration)

1. **セマンティックバージョニングの尊重**:
   - 仕様変更に伴う破壊的変更（例: コマンドライン引数の廃止、出力パスルールの変更等）は、メジャーバージョンアップ（例: `v2.0.0`）でのみ導入されます。
2. **非推奨（Deprecation）期間の提供**:
   - 既存機能を変更または非推奨とする場合、最低 1 つのマイナーリリース期間において警告メッセージを表示し、段階的な移行期間を設けます。
3. **新フォーマットバージョン (`v2`) への移行**:
   - 入力フォーマットに非互換な新機能が追加される場合は、frontmatter 内のバージョン指定（例: `uchi: v2`）を新設します。
   - `v1` 解析器は `uchi: v1` 宣言文書のみを処理するため、新旧バージョンのドキュメントは安全に共存・遷移が可能です。

---

## 2. 回帰テストとしてのゴールデン fixture (Golden Fixture Testing)

`uchi` では、既存の `v1` 生成結果の後退（エンバグ）を自動検出するための回帰テストとしてゴールデン fixture（Golden Test Fixtures）を運用しています。

### 構成と検証メカニズム

- **データ配置**: `internal/app/testdata/<fixture_name>/`
  - `input/`: 入力 Markdown ファイル群
  - `want/`: 期待される抽出・生成結果ディレクトリ構造およびバイト内容
- **自動検証**:
  - `go test ./internal/app` 実行時、テストスイート (`TestGolden`) が `input` を一時ディレクトリにコピーし、`app.Run` を実行します。
  - 生成された出力ディレクトリと `want` ディレクトリの間で、ファイルの過不足および内容の差分（行単位 diff）が自動比較検証されます。
- **ゴールデン更新手順**:
  - 仕様変更等により意図的に期待結果を更新する場合は、以下のコマンドを明示的に実行します。

```bash
go test ./internal/app -update
```

---

## 3. 性能基準と入力規模 (Performance Criteria)

`uchi` は単一の Go バイナリとして動作する軽量ツールであり、不要な並列化や外部ベンチマークインフラを導入せず、ローカル環境で迅速かつ決定論的に完了する性能基準を維持します。

### 想定入力規模 (Standard Input Scale)

- **小〜中規模 dotfiles リポジトリ**:
  - ドキュメント数: 10 〜 100 ファイル
  - 総行数: 数千行 〜 10,000 行程度
  - コードブロック数: 50 〜 500 ブロック

### 性能基準 (Target Performance)

- **解析・計画構築 (`check` / `dry-run`)**: 100 ファイル規模の入力に対し **50 ms 以下**
- **ファイル抽出・生成 (`gen`)**: 100 ファイル規模の入力に対し **200 ms 以下**（アトミック書き込み含む）
- **メモリ消費量**: 大規模な入力であってもピーク時 **50 MB 以下** に抑え、リソースを過剰に浪費しないこと。

---

## 4. 性能および堅牢性の計測手順 (Measurement Procedures)

開発者および CI 環境において、性能変化や堅牢性を検証するための標準的な計測手順です。

### ベンチマークテストの実行

Go 標準のベンチマーク機能を用いて、パーサーおよび生成処理の処理速度（`ns/op`）を計測します。

```bash
# 全パッケージのベンチマークを実行
go test -bench=. ./...

# Markdown パーサーの個別ベンチマーク
go test -bench=BenchmarkParseFile ./internal/markdown

# 生成計画構築の個別ベンチマーク
go test -bench=BenchmarkBuildGenerationPlan ./internal/app
```

### Fuzz テスト（堅牢性・入力境界の検証）

Markdown frontmatter および属性解析器に対する fuzz テストを実行し、クラッシュ、パニック、無限ループ、異常なメモリ割り当てがないか検証します。

```bash
# frontmatter パーサーの Fuzz テスト (5秒間)
go test -fuzz=FuzzParseFrontmatter -fuzztime=5s ./internal/markdown

# 属性・コードフェンスヘッダーの Fuzz テスト (5秒間)
go test -fuzz=FuzzParseFenceHeader -fuzztime=5s ./internal/markdown

# ファイル解析全体の Fuzz テスト (5秒間)
go test -fuzz=FuzzParseFile -fuzztime=5s ./internal/markdown
```

### CLI 実行時間の直接計測

ビルドされた CLI バイナリの実行時間を直接計測する場合の手順です。

```bash
# ビルド
go build -o uchi ./cmd/uchi

# check コマンドの実行時間計測
time ./uchi check

# gen コマンドの実行時間計測
time ./uchi gen -o ./dist
```
