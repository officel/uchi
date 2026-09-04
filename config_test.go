package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InputDir != "./toc" {
		t.Errorf("expected InputDir to be './toc', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "./dist" {
		t.Errorf("expected OutputDir to be './dist', got %s", cfg.OutputDir)
	}
}

func TestLoadConfigFlags(t *testing.T) {
	args := []string{"-i", "input_dir", "-o", "output_dir"}
	cfg, err := LoadConfig(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InputDir != "input_dir" {
		t.Errorf("expected InputDir to be 'input_dir', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "output_dir" {
		t.Errorf("expected OutputDir to be 'output_dir', got %s", cfg.OutputDir)
	}
}

func TestLoadConfigFileAndFlagsOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configData := `{"input_dir": "json_in", "output_dir": "json_out"}`

	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test config file loading without overrides
	cfg, err := LoadConfig([]string{"-c", configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InputDir != "json_in" {
		t.Errorf("expected InputDir 'json_in', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "json_out" {
		t.Errorf("expected OutputDir 'json_out', got %s", cfg.OutputDir)
	}

	// Test CLI flag overrides config file
	cfgOverride, err := LoadConfig([]string{"-c", configPath, "-o", "flag_out"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfgOverride.InputDir != "json_in" {
		t.Errorf("expected InputDir 'json_in', got %s", cfgOverride.InputDir)
	}
	if cfgOverride.OutputDir != "flag_out" {
		t.Errorf("expected OutputDir 'flag_out', got %s", cfgOverride.OutputDir)
	}
}
