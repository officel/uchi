package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/officel/uchi/internal/config"
)

func TestBuildAndVerifyGenerationPlan(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	gitMdPath := filepath.Join(inputDir, "git.md")
	gitMdContent := "---\nuchi: v1\n---\n## Environment\n\n```sh {schema=env}\nGIT_PAGER=vim\n```\n\n## Alias\n\n```sh {schema=alias}\nalias g=\"git\"\n```\n"
	if err := os.WriteFile(gitMdPath, []byte(gitMdContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		InputDir:    inputDir,
		OutputDir:   outputDir,
		AutoComment: true,
		Command:     "gen",
	}

	plan, err := buildGenerationPlan(cfg)
	if err != nil {
		t.Fatalf("buildGenerationPlan() unexpected error = %v", err)
	}

	// Verify output directory is NOT created during plan construction
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("expected output directory %s to not exist after buildGenerationPlan, got err = %v", outputDir, err)
	}

	// Verify Documents
	if len(plan.Documents) != 1 {
		t.Fatalf("len(plan.Documents) = %d, want 1", len(plan.Documents))
	}

	doc := plan.Documents[0]
	if doc.SourcePath != gitMdPath {
		t.Errorf("doc.SourcePath = %q, want %q", doc.SourcePath, gitMdPath)
	}
	if doc.RelativeBase != "git" {
		t.Errorf("doc.RelativeBase = %q, want %q", doc.RelativeBase, "git")
	}
	if len(doc.Snippets) != 2 {
		t.Fatalf("len(doc.Snippets) = %d, want 2", len(doc.Snippets))
	}

	// Snippet 1: env at line 6
	snip0 := doc.Snippets[0]
	if snip0.Schema != "env" {
		t.Errorf("snip0.Schema = %q, want %q", snip0.Schema, "env")
	}
	if snip0.Source.Path != gitMdPath || snip0.Source.Line != 6 {
		t.Errorf("snip0.Source = %v, want %s:6", snip0.Source, gitMdPath)
	}
	if snip0.ProcessedContent != "GIT_PAGER=vim" {
		t.Errorf("snip0.ProcessedContent = %q, want %q", snip0.ProcessedContent, "GIT_PAGER=vim")
	}

	// Snippet 2: alias at line 12
	snip1 := doc.Snippets[1]
	if snip1.Schema != "alias" {
		t.Errorf("snip1.Schema = %q, want %q", snip1.Schema, "alias")
	}
	if snip1.Source.Path != gitMdPath || snip1.Source.Line != 12 {
		t.Errorf("snip1.Source = %v, want %s:12", snip1.Source, gitMdPath)
	}
	if snip1.ProcessedContent != "alias g=\"git\"" {
		t.Errorf("snip1.ProcessedContent = %q, want %q", snip1.ProcessedContent, "alias g=\"git\"")
	}

	// Verify Targets
	if len(plan.Targets) != 4 {
		t.Fatalf("len(plan.Targets) = %d, want 4 (2 parts, 2 merged)", len(plan.Targets))
	}

	// Target 0: part git env
	t0 := plan.Targets[0]
	if t0.Kind != TargetKindPart {
		t.Errorf("t0.Kind = %q, want %q", t0.Kind, TargetKindPart)
	}
	if t0.Schema != "env" {
		t.Errorf("t0.Schema = %q, want %q", t0.Schema, "env")
	}
	if !strings.HasSuffix(t0.Path, filepath.Join("parts", "git", "env")) {
		t.Errorf("t0.Path = %q, want suffix %q", t0.Path, filepath.Join("parts", "git", "env"))
	}
	if string(t0.Content) != "# git\nGIT_PAGER=vim\n" {
		t.Errorf("t0.Content = %q, want %q", string(t0.Content), "# git\nGIT_PAGER=vim\n")
	}
	if len(t0.Sources) != 1 || t0.Sources[0].Line != 6 {
		t.Errorf("t0.Sources = %v, want [{%s 6}]", t0.Sources, gitMdPath)
	}

	// Target 1: part git alias
	t1 := plan.Targets[1]
	if t1.Kind != TargetKindPart {
		t.Errorf("t1.Kind = %q, want %q", t1.Kind, TargetKindPart)
	}
	if t1.Schema != "alias" {
		t.Errorf("t1.Schema = %q, want %q", t1.Schema, "alias")
	}
	if string(t1.Content) != "# git\nalias g=\"git\"\n" {
		t.Errorf("t1.Content = %q, want %q", string(t1.Content), "# git\nalias g=\"git\"\n")
	}

	// Target 2: merged env
	t2 := plan.Targets[2]
	if t2.Kind != TargetKindMerged {
		t.Errorf("t2.Kind = %q, want %q", t2.Kind, TargetKindMerged)
	}
	if t2.Schema != "env" {
		t.Errorf("t2.Schema = %q, want %q", t2.Schema, "env")
	}
	if string(t2.Content) != "# git\nGIT_PAGER=vim\n" {
		t.Errorf("t2.Content = %q, want %q", string(t2.Content), "# git\nGIT_PAGER=vim\n")
	}

	// Target 3: merged alias
	t3 := plan.Targets[3]
	if t3.Kind != TargetKindMerged {
		t.Errorf("t3.Kind = %q, want %q", t3.Kind, TargetKindMerged)
	}
	if t3.Schema != "alias" {
		t.Errorf("t3.Schema = %q, want %q", t3.Schema, "alias")
	}
	if string(t3.Content) != "# git\nalias g=\"git\"\n" {
		t.Errorf("t3.Content = %q, want %q", string(t3.Content), "# git\nalias g=\"git\"\n")
	}

	// Verify plan validation passes
	if err := verifyGenerationPlan(plan); err != nil {
		t.Fatalf("verifyGenerationPlan() error = %v", err)
	}
}

func TestBuildGenerationPlanDeterministicOrder(t *testing.T) {
	dir := t.TempDir()

	fileSpecs := map[string]string{
		"z.md":     "---\nuchi: v1\n---\n```sh {schema=alias}\nalias z=\"z\"\n```\n",
		"a.md":     "---\nuchi: v1\n---\n```sh {schema=env}\nA=1\n```\n",
		"sub/b.md": "---\nuchi: v1\n---\n```sh {schema=alias}\nalias b=\"b\"\n```\n",
	}

	input1 := filepath.Join(dir, "input1")
	out1 := filepath.Join(dir, "dist1")
	for _, name := range []string{"z.md", "sub/b.md", "a.md"} {
		p := filepath.Join(input1, name)
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		_ = os.WriteFile(p, []byte(fileSpecs[name]), 0644)
	}

	input2 := filepath.Join(dir, "input2")
	out2 := filepath.Join(dir, "dist2")
	for _, name := range []string{"a.md", "z.md", "sub/b.md"} {
		p := filepath.Join(input2, name)
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		_ = os.WriteFile(p, []byte(fileSpecs[name]), 0644)
	}

	plan1, err1 := buildGenerationPlan(&config.Config{InputDir: input1, OutputDir: out1, AutoComment: true})
	plan2, err2 := buildGenerationPlan(&config.Config{InputDir: input2, OutputDir: out2, AutoComment: true})

	if err1 != nil || err2 != nil {
		t.Fatalf("buildGenerationPlan errors: %v, %v", err1, err2)
	}

	if len(plan1.Documents) != len(plan2.Documents) {
		t.Fatalf("len(plan1.Documents) = %d, len(plan2.Documents) = %d", len(plan1.Documents), len(plan2.Documents))
	}

	for i := range plan1.Documents {
		if plan1.Documents[i].RelativeBase != plan2.Documents[i].RelativeBase {
			t.Errorf("doc[%d] RelativeBase mismatch: %q vs %q", i, plan1.Documents[i].RelativeBase, plan2.Documents[i].RelativeBase)
		}
	}

	if len(plan1.Targets) != len(plan2.Targets) {
		t.Fatalf("len(plan1.Targets) = %d, len(plan2.Targets) = %d", len(plan1.Targets), len(plan2.Targets))
	}

	for i := range plan1.Targets {
		t1 := plan1.Targets[i]
		t2 := plan2.Targets[i]
		if t1.RelativePath != t2.RelativePath {
			t.Errorf("target[%d] RelativePath mismatch: %q vs %q", i, t1.RelativePath, t2.RelativePath)
		}
		if t1.Kind != t2.Kind {
			t.Errorf("target[%d] Kind mismatch: %q vs %q", i, t1.Kind, t2.Kind)
		}
		if string(t1.Content) != string(t2.Content) {
			t.Errorf("target[%d] Content mismatch:\nplan1:\n%s\nplan2:\n%s", i, string(t1.Content), string(t2.Content))
		}
	}
}

func TestVerifyGenerationPlanErrors(t *testing.T) {
	t.Run("returns error for nil plan", func(t *testing.T) {
		err := verifyGenerationPlan(nil)
		if err == nil {
			t.Fatal("expected error for nil plan, got nil")
		}
	})

	t.Run("returns error for empty target path", func(t *testing.T) {
		plan := &GenerationPlan{
			OutputDir: "/tmp",
			Targets: []TargetFile{
				{Path: ""},
			},
		}
		err := verifyGenerationPlan(plan)
		if err == nil {
			t.Fatal("expected error for empty target path, got nil")
		}
	})

	t.Run("returns error for conflicting target paths", func(t *testing.T) {
		plan := &GenerationPlan{
			OutputDir: "/tmp",
			Targets: []TargetFile{
				{Path: "/tmp/alias", Schema: "alias"},
				{Path: "/tmp/alias", Schema: "alias_duplicate"},
			},
		}
		err := verifyGenerationPlan(plan)
		if err == nil {
			t.Fatal("expected error for duplicate target path, got nil")
		}
		if !strings.Contains(err.Error(), "conflicting target file path") {
			t.Errorf("err = %q, want conflict message", err.Error())
		}
	})
}
