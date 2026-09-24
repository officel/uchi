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
	switch cfg.Command {
	case "gen":
		return runExtraction(cfg)
	case "new":
		return runNew(cfg, output)
	case "init":
		return runInit(cfg, output)
	case "", "check":
		return runCheck(cfg, output)
	default:
		return fmt.Errorf("unknown command '%s'", cfg.Command)
	}
}

func validateInputs(cfg *config.Config) ([]markdown.Document, error) {
	if _, err := os.Stat(cfg.InputDir); err != nil {
		return nil, fmt.Errorf("failed to access input directory %s: %w", cfg.InputDir, err)
	}

	documents, err := markdown.Walk(cfg.InputDir)
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func runCheck(cfg *config.Config, output io.Writer) error {
	if _, err := validateInputs(cfg); err != nil {
		return err
	}

	if cfg.ConfigFile != "" {
		fmt.Fprintf(output, "Config file: found (%s)\n", cfg.ConfigFile)
	} else {
		fmt.Fprintln(output, "Config file: not found")
	}
	fmt.Fprintln(output, "Options:")
	fmt.Fprintf(output, "  input_dir: %s\n", cfg.InputDir)
	fmt.Fprintf(output, "  output_dir: %s\n", cfg.OutputDir)
	fmt.Fprintf(output, "  template_dir: %s\n", cfg.TemplateDir)
	fmt.Fprintf(output, "  auto_comment: %t\n", cfg.AutoComment)
	return nil
}

type outputFile struct {
	path    string
	content []byte
}

func buildOutputPath(outputDir string, elements ...string) (string, error) {
	for _, elem := range elements {
		trimmed := strings.TrimSpace(elem)
		if trimmed == "" {
			return "", fmt.Errorf("empty path element")
		}
		if filepath.IsAbs(elem) || strings.HasPrefix(elem, "/") || strings.HasPrefix(elem, "\\") {
			return "", fmt.Errorf("absolute path element %q is not allowed", elem)
		}
		parts := strings.FieldsFunc(elem, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		for _, part := range parts {
			if part == ".." {
				return "", fmt.Errorf("path element %q contains '..' segment", elem)
			}
		}
		base := filepath.Base(elem)
		if base == "." || base == ".." || base == "/" || base == "\\" || strings.TrimSpace(base) == "" {
			return "", fmt.Errorf("invalid or empty base name in path element %q", elem)
		}
	}

	subPath := filepath.Join(elements...)
	fullPath := filepath.Join(outputDir, subPath)

	absOutDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for output dir %s: %w", outputDir, err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for output path %s: %w", fullPath, err)
	}

	rel, err := filepath.Rel(absOutDir, absFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path from output dir: %w", err)
	}

	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "..\\") {
		return "", fmt.Errorf("output path %q escapes output directory %q", fullPath, outputDir)
	}

	return fullPath, nil
}

func runExtraction(cfg *config.Config) error {
	if err := os.MkdirAll(cfg.InputDir, 0755); err != nil {
		return fmt.Errorf("failed to create input directory %s: %w", cfg.InputDir, err)
	}

	documents, err := validateInputs(cfg)
	if err != nil {
		return err
	}

	var filesToWrite []outputFile
	mergedSchemaContents := make(map[string][]string)

	for _, document := range documents {
		if document.UchiVersion != "v1" {
			continue
		}

		relativePath, err := filepath.Rel(cfg.InputDir, document.FilePath)
		if err != nil || strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
			return &markdown.Diagnostic{
				Path:    document.FilePath,
				Message: fmt.Sprintf("invalid input relative path %q", relativePath),
			}
		}

		ext := filepath.Ext(relativePath)
		relBase := strings.TrimSuffix(relativePath, ext)
		baseStem := filepath.Base(relBase)
		if strings.TrimSpace(relBase) == "" || strings.HasSuffix(relBase, "/") || strings.HasSuffix(relBase, "\\") || baseStem == "." || baseStem == ".." || strings.TrimSpace(baseStem) == "" {
			return &markdown.Diagnostic{
				Path:    document.FilePath,
				Message: fmt.Sprintf("invalid output base name for document %q", document.FilePath),
			}
		}

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
			outPath, err := buildOutputPath(cfg.OutputDir, "parts", relBase, schemaName)
			if err != nil {
				return &markdown.Diagnostic{
					Path:    document.FilePath,
					Message: fmt.Sprintf("invalid output path for parts: %v", err),
				}
			}

			var nonEmpty []string
			for _, c := range contents {
				if strings.TrimSpace(c) != "" {
					nonEmpty = append(nonEmpty, c)
				}
			}
			fileCombined := strings.Join(nonEmpty, "\n")
			if cfg.AutoComment && len(nonEmpty) > 0 {
				toolName := filepath.ToSlash(relBase)
				fileCombined = "# " + toolName + "\n" + fileCombined
			}
			data := []byte(formatOutput(fileCombined))
			filesToWrite = append(filesToWrite, outputFile{path: outPath, content: data})

			mergedSchemaContents[schemaName] = append(mergedSchemaContents[schemaName], fileCombined)
		}
	}

	for schemaName, contents := range mergedSchemaContents {
		outPath, err := buildOutputPath(cfg.OutputDir, schemaName)
		if err != nil {
			return fmt.Errorf("invalid output path for merged schema %s: %w", schemaName, err)
		}

		var nonEmpty []string
		for _, c := range contents {
			if strings.TrimSpace(c) != "" {
				nonEmpty = append(nonEmpty, c)
			}
		}
		mergedCombined := strings.Join(nonEmpty, "\n\n")
		data := []byte(formatOutput(mergedCombined))
		filesToWrite = append(filesToWrite, outputFile{path: outPath, content: data})
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", cfg.OutputDir, err)
	}

	for _, file := range filesToWrite {
		if err := os.MkdirAll(filepath.Dir(file.path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(file.path, file.content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.path, err)
		}
	}

	return nil
}

func formatOutput(content string) string {
	if strings.TrimSpace(content) == "" {
		return "\n"
	}
	return strings.TrimRight(content, "\r\n") + "\n"
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
