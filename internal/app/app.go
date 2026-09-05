package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/officel/uchi/internal/config"
	"github.com/officel/uchi/internal/markdown"
	"github.com/officel/uchi/internal/template"
)

// Run executes the configured command.
func Run(cfg *config.Config, output io.Writer) error {
	if cfg.Command == "new" {
		return runNew(cfg, output)
	}
	if cfg.Command == "init" {
		return runInit(cfg, output)
	}
	return runExtraction(cfg)
}

func runExtraction(cfg *config.Config) error {
	if err := os.MkdirAll(cfg.InputDir, 0755); err != nil {
		return fmt.Errorf("failed to create input directory %s: %w", cfg.InputDir, err)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", cfg.OutputDir, err)
	}

	documents, err := markdown.Walk(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to process directory %s: %w", cfg.InputDir, err)
	}
	for _, document := range documents {
		if !strings.Contains(document.Frontmatter, "uchi: v1") {
			continue
		}
		if err := writeCodeFences(cfg.InputDir, cfg.OutputDir, document); err != nil {
			return err
		}
	}
	return nil
}

func writeCodeFences(inputDir, outputDir string, document markdown.Document) error {
	if len(document.CodeFences) == 0 {
		return nil
	}

	relativePath, err := filepath.Rel(inputDir, document.FilePath)
	if err != nil {
		relativePath = filepath.Base(document.FilePath)
	}
	base := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))

	for index, fence := range document.CodeFences {
		filename := base + ".txt"
		if len(document.CodeFences) > 1 {
			filename = fmt.Sprintf("%s_%d.txt", base, index+1)
		}
		path := filepath.Join(outputDir, filename)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(fence.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}
	return nil
}

func runNew(cfg *config.Config, output io.Writer) error {
	if cfg.CommandArg == "" {
		return fmt.Errorf("new subcommand requires a target name")
	}
	if err := os.MkdirAll(cfg.InputDir, 0755); err != nil {
		return fmt.Errorf("failed to create input directory %s: %w", cfg.InputDir, err)
	}

	content, err := template.NewResolver(cfg.TemplateDir).Render("new.md", template.Data{
		Name: cfg.CommandArg,
		Date: time.Now().Format("2006-01-02"),
	})
	if err != nil {
		return err
	}

	filename := cfg.CommandArg
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}
	targetPath := filepath.Join(cfg.InputDir, filename)
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	_, err = fmt.Fprintf(output, "Created %s\n", targetPath)
	return err
}

func runInit(cfg *config.Config, output io.Writer) error {
	targetPath := cfg.ConfigFile
	if targetPath == "" {
		targetPath = config.DefaultConfigPaths[0]
	}
	if err := config.Save(targetPath, cfg); err != nil {
		return fmt.Errorf("failed to create configuration file %s: %w", targetPath, err)
	}
	_, err := fmt.Fprintf(output, "Created %s\n", targetPath)
	return err
}
