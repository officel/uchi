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
	"github.com/officel/uchi/internal/schema"
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

	mergedSchemaContents := make(map[string][]string)

	for _, document := range documents {
		if !strings.Contains(document.Frontmatter, "uchi: v1") {
			continue
		}

		relativePath, err := filepath.Rel(cfg.InputDir, document.FilePath)
		if err != nil {
			relativePath = filepath.Base(document.FilePath)
		}
		relBase := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))

		fileSchemaContents := make(map[string][]string)

		for _, fence := range document.CodeFences {
			if !fence.HasAnnotation {
				continue
			}
			schemaName := fence.Schema()
			if schemaName == "" {
				continue
			}
			processed, ok := schema.Process(schemaName, fence.Content)
			if !ok {
				continue
			}

			fileSchemaContents[schemaName] = append(fileSchemaContents[schemaName], processed)
		}

		for schemaName, contents := range fileSchemaContents {
			outPath := filepath.Join(cfg.OutputDir, relBase, schemaName)
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}
			fileCombined := strings.Join(contents, "\n")
			data := []byte(fileCombined)
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", outPath, err)
			}
			mergedSchemaContents[schemaName] = append(mergedSchemaContents[schemaName], fileCombined)
		}
	}

	for schemaName, contents := range mergedSchemaContents {
		outPath := filepath.Join(cfg.OutputDir, schemaName)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		data := []byte(strings.Join(contents, "\n\n"))
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write merged file %s: %w", outPath, err)
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
