package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseFrontmatter(f *testing.F) {
	seeds := []string{
		"uchi: v1\ntitle: test",
		"title: General Note",
		"",
		"uchi: v2",
		"uchi: 123",
		"uchi: [invalid yaml",
		"uchi: \"v1\"\nkey: val",
		"   uchi: v1  ",
		"# uchi: v1\ntitle: test",
		"uchi:\n  - v1",
		"uchi:\n  version: v1",
		"uchi: null",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// ParseFrontmatter should handle any string input without panicking.
		_, _ = ParseFrontmatter(input)
	})
}

func FuzzParseFenceHeader(f *testing.F) {
	seeds := []string{
		"```bash",
		"```",
		"```sh {schema=env}",
		"```sh {schema=alias flag}",
		"```bash {schema=\"env with space\"}",
		"```zsh {schema='env with single space'}",
		"```{schema=rc}",
		"```sh {schema=env schema=alias}",
		"```sh {schema=}",
		"```sh {schema=\"\"}",
		"```sh {schema=''}",
		"```sh {schema=env",
		"```sh schema=env}",
		"```sh {schema=env} extra",
		"```sh {schema=\"env}",
		"```sh {schema='env}",
		"```sh {=env}",
		"```sh {schema?=env}",
		"```sh {schema=\"env\"extra}",
		"```sh {schema=alias target=bash}",
		"```sh {schema=alias target=\"bash, zsh\"}",
		"```sh {schema=env target=\"all, bash\"}",
		"```sh {schema=env target=\",\"}",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// ParseFenceHeader should parse code fence headers deterministically without panicking.
		_, _ = ParseFenceHeader(input)
	})
}

func FuzzParseFile(f *testing.F) {
	seeds := [][]byte{
		[]byte("---\nuchi: v1\n---\n\n```sh {schema=env}\nFOO=bar\n```\n"),
		[]byte("---\nuchi: v1\n---\n\n```sh {schema=env target=bash}\nexport FOO=bar\n```\n"),
		[]byte("---\ntitle: Note\n---\n```sh\necho hi\n```\n"),
		[]byte("---\nuchi: [broken\n---\n"),
		[]byte("---\nuchi: v1\n---\n```sh {schema=env\nFOO=bar\n```\n"),
		[]byte("---\nuchi: v1\n---\n```sh {schema=unknown}\n```\n"),
		[]byte("---\nuchi: v1\n---\n```sh {schema=alias target=\"bash, bash\"}\n```\n"),
		[]byte("```sh {schema=alias}\nalias a=b\n```\n"),
		[]byte(""),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Prevent excessive memory allocation for abnormally large fuzz inputs
		if len(data) > 100000 {
			return
		}

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "fuzz.md")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			t.Skip()
		}

		// ParseFile should parse arbitrary file bytes safely without panicking.
		_, _ = ParseFile(filePath)
	})
}
