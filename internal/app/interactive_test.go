package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/officel/uchi/internal/config"
	"github.com/officel/uchi/internal/markdown"
)

func TestInteractiveRecoveryUnknownSchemaConfirmed(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(inputDir, "bad_schema.md")
	badContent := "---\nuchi: v1\n---\n```sh {schema=invalid_schema}\nalias foo='bar'\n```\n"
	if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Simulated user input:
	// 1) Select '1' (view fix suggestion)
	// 2) Select '2' (apply fix)
	// 3) Select '1' (choose alias schema)
	// 4) Select 'y' (confirm apply)
	mockStdin := strings.NewReader("1\n2\n1\ny\n")
	var outBuf bytes.Buffer

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		Command:     "check",
		Interactive: true,
		Stdin:       mockStdin,
	}

	err := Run(cfg, &outBuf)
	if err != nil {
		t.Fatalf("expected check --interactive to succeed after auto-fix, got err = %v", err)
	}

	// Verify file content was fixed
	fixedData, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fixedData), "{schema=alias}") {
		t.Errorf("file content was not updated to alias schema: %s", string(fixedData))
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "[interactive]") {
		t.Errorf("output missing interactive header: %s", outStr)
	}
	if !strings.Contains(outStr, "修正を適用しました") {
		t.Errorf("output missing fix completion message: %s", outStr)
	}
}

func TestInteractiveRecoveryUnknownSchemaDeclined(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(inputDir, "bad_schema.md")
	badContent := "---\nuchi: v1\n---\n```sh {schema=invalid_schema}\nalias foo='bar'\n```\n"
	if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	// User selects apply fix, chooses alias schema, but declines confirmation with 'n'
	mockStdin := strings.NewReader("2\n1\nn\n3\n")
	var outBuf bytes.Buffer

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		Command:     "check",
		Interactive: true,
		Stdin:       mockStdin,
	}

	err := Run(cfg, &outBuf)
	if err == nil {
		t.Fatal("expected check --interactive to fail when fix is declined, got nil")
	}

	// Verify file content was NOT changed
	unfixedData, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(unfixedData) != badContent {
		t.Errorf("file content was altered despite user decline: %s", string(unfixedData))
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "修正の適用をキャンセルしました") {
		t.Errorf("output missing cancellation message: %s", outStr)
	}
}

func TestInteractiveRecoveryUnknownTargetShellConfirmed(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(inputDir, "bad_shell.md")
	badContent := "---\nuchi: v1\n---\n```sh {schema=alias target=unknown_shell}\nalias foo='bar'\n```\n"
	if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	// User input:
	// 1) Select '2' (apply fix)
	// 2) Select '2' (choose 'bash' target)
	// 3) Select 'y' (confirm)
	mockStdin := strings.NewReader("2\n2\ny\n")
	var outBuf bytes.Buffer

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		Command:     "check",
		Interactive: true,
		Stdin:       mockStdin,
	}

	err := Run(cfg, &outBuf)
	if err != nil {
		t.Fatalf("expected check --interactive to succeed after target shell fix, got err = %v", err)
	}

	fixedData, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fixedData), "target=bash") {
		t.Errorf("file content missing target=bash: %s", string(fixedData))
	}
}

func TestInteractiveRecoveryBrokenFrontmatterConfirmed(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(inputDir, "bad_fm.md")
	badContent := "---\nuchi: invalid_version\n---\n```sh {schema=alias}\nalias foo='bar'\n```\n"
	if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	// User input:
	// 1) Select '2' (apply fix)
	// 2) Select 'y' (confirm frontmatter insertion)
	mockStdin := strings.NewReader("2\ny\n")
	var outBuf bytes.Buffer

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		Command:     "check",
		Interactive: true,
		Stdin:       mockStdin,
	}

	err := Run(cfg, &outBuf)
	if err != nil {
		t.Fatalf("expected check --interactive to succeed after frontmatter fix, got err = %v", err)
	}

	fixedData, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(fixedData), "---\nuchi: v1\n---") {
		t.Errorf("file content missing uchi: v1 frontmatter: %s", string(fixedData))
	}
}

func TestInteractiveRecoveryNonInteractiveFallback(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(inputDir, "bad.md")
	badContent := "---\nuchi: v1\n---\n```sh {schema=invalid_schema}\nalias foo='bar'\n```\n"
	if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create InteractiveSession where IsTerminal is false
	var outBuf bytes.Buffer
	sess := &InteractiveSession{
		Reader:     strings.NewReader(""),
		Writer:     &outBuf,
		IsTerminal: false,
	}

	origDiag := &markdown.Diagnostic{Path: badPath, Line: 4, Message: "unknown schema \"invalid_schema\""}
	err := sess.RunInteractiveRecovery(origDiag)

	if err != origDiag {
		t.Errorf("expected RunInteractiveRecovery to return origDiag on non-terminal, got %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "非対話環境") {
		t.Errorf("output missing non-interactive environment warning: %s", outStr)
	}
}
