# uchi

Dotfiles Literate Configuration CLI tool.

## uchi is

- このアプリケーションは、dotfiles で管理するツールの設定等をマークダウンで管理します
- リテレートコンフィギュレーション（Literate Configuration）の思想に基づいています
- 管理するツール（インストールしたツール）の設定をマークダウンで管理し、所定のコードフェンスから統合された設定ファイルを生成します
- 既存の dotfiles 管理や chezmoi 等のツールとバッティングしないように、マークダウンファイルのスキーマ管理と設定の出力のみを行います
- input ディレクトリ（`./toc/`）に所定のマークダウンファイルを配置します
- output ディレクトリ（`./dist/`）に設定を書き出します
- 書き出しの際、uchi の実行シェルに応じて自動的に出力を出しわける予定です
- `bash` による書式をベースに、`zsh` や `PowerShell` 等への書式変換を行う予定だということです
- Chez moi （シェモア）はフランス語で「私の家（で）」という意味だそうです。uchi はもちろん、うち（私の家）という命名です

## Build & Usage

Build, test, and template usage are documented in [BUILD.md](BUILD.md).
