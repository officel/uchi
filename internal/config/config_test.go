package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsAndAutoCreate(t *testing.T) {
	changeToTempDir(t)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InputDir != "./toc" || cfg.OutputDir != "./dist" {
		t.Fatalf("Load() = %+v, want default input and output directories", cfg)
	}

	content, err := os.ReadFile(".uchi.yaml")
	if err != nil {
		t.Fatalf("expected .uchi.yaml to be created: %v", err)
	}
	if strings.Contains(string(content), "config_file") {
		t.Errorf("configuration unexpectedly contains config_file: %s", content)
	}
}

func TestLoadUsesDotConfigFile(t *testing.T) {
	changeToTempDir(t)
	if err := os.MkdirAll(".config", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".config", ".uchi.yaml"), []byte("input_dir: yaml_in\noutput_dir: yaml_out\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InputDir != "yaml_in" || cfg.OutputDir != "yaml_out" {
		t.Errorf("Load() = %+v, want YAML values", cfg)
	}
	if _, err := os.Stat(".uchi.yaml"); !os.IsNotExist(err) {
		t.Error(".uchi.yaml was created despite .config/.uchi.yaml existing")
	}
}

func TestLoadFlagsOverrideFileAndSupportTemplateDirectory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("input_dir: yaml_in\noutput_dir: yaml_out\ntemplate_dir: yaml_templates\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load([]string{"-c", configPath, "--output", "flag_out", "-t", "flag_templates"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InputDir != "yaml_in" || cfg.OutputDir != "flag_out" || cfg.TemplateDir != "flag_templates" {
		t.Errorf("Load() = %+v, want YAML input plus flag overrides", cfg)
	}
}

func TestLoadValidatesFlagSyntaxAndNewCommand(t *testing.T) {
	for _, args := range [][]string{{"-input", "dir"}, {"--i", "dir"}, {"-config", "file.yaml"}, {"--c", "file.yaml"}} {
		if _, err := Load(args); err == nil {
			t.Errorf("Load(%v) error = nil, want syntax error", args)
		}
	}

	changeToTempDir(t)
	cfg, err := Load([]string{"-i", "custom_toc", "new", "git"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Command != "new" || cfg.CommandArg != "git" || cfg.InputDir != "custom_toc" {
		t.Errorf("Load() = %+v, want parsed new command", cfg)
	}
}

func changeToTempDir(t *testing.T) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
