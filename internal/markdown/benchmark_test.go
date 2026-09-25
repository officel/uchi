package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkParseFrontmatter(b *testing.B) {
	input := "uchi: v1\ntitle: Benchmark Document\ndescription: Performance testing frontmatter"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseFrontmatter(input)
	}
}

func BenchmarkParseFenceHeader(b *testing.B) {
	input := "```sh {schema=env target=\"bash, zsh\"}"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseFenceHeader(input)
	}
}

func BenchmarkParseFile(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.md")
	content := "---\nuchi: v1\ntitle: Benchmark Document\n---\n\n# Section 1\n\n```sh {schema=env target=bash}\nexport PATH=\"$HOME/bin:$PATH\"\nexport EDITOR=vim\n```\n\n```sh {schema=alias target=\"bash,zsh\"}\nalias ll=\"ls -la\"\nalias g=\"git\"\n```\n\n```sh {schema=rc}\nset -o vi\n```\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseFile(path)
	}
}
