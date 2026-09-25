package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/officel/uchi/internal/config"
)

func BenchmarkBuildGenerationPlan(b *testing.B) {
	dir := b.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		doc := fmt.Sprintf("---\nuchi: v1\ntitle: Document %d\n---\n\n```sh {schema=env}\nENV_%d=val%d\n```\n\n```sh {schema=alias}\nalias a%d=\"cmd%d\"\n```\n", i, i, i, i, i)
		if err := os.WriteFile(filepath.Join(inputDir, fmt.Sprintf("doc_%d.md", i)), []byte(doc), 0644); err != nil {
			b.Fatal(err)
		}
	}

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		AutoComment: true,
		Command:     "check",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := buildGenerationPlan(cfg, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunGen(b *testing.B) {
	dir := b.TempDir()
	inputDir := filepath.Join(dir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		doc := fmt.Sprintf("---\nuchi: v1\n---\n\n```sh {schema=env}\nVAR_%d=val\n```\n\n```sh {schema=alias}\nalias a%d=\"echo %d\"\n```\n", i, i, i)
		if err := os.WriteFile(filepath.Join(inputDir, fmt.Sprintf("doc_%d.md", i)), []byte(doc), 0644); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputDir := filepath.Join(dir, fmt.Sprintf("dist_%d", i))
		cfg := &config.Config{
			InputDir:    inputDir,
			OutputDir:   outputDir,
			AutoComment: true,
			Command:     "gen",
		}
		var buf bytes.Buffer
		if err := Run(cfg, &buf); err != nil {
			b.Fatal(err)
		}
	}
}
