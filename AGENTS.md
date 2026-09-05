# AGENTS

- Go CLI プロジェクト
- 実行入口: cmd/uchi
- アプリ本体: internal/
- 変更は小さく読みやすく
- Go 標準ライブラリと現在の構成を優先
- テスト: `go test ./...`
- ビルド: `go build ./cmd/uchi`
- 終了前に `gofmt` を実行
- 終了前に `prek run -a` を実行し、エラーに対応すること
- コミットタイトルは Conventional Commit に従い、英語で記述すること
- コミット本文、PRの本文は日本語で詳細な説明を記述すること
