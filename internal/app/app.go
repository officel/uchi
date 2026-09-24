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

func buildGenerationPlan(cfg *config.Config) (*GenerationPlan, error) {
	if err := os.MkdirAll(cfg.InputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create input directory %s: %w", cfg.InputDir, err)
	}

	documents, err := validateInputs(cfg)
	if err != nil {
		return nil, err
	}

	plan := &GenerationPlan{
		OutputDir: cfg.OutputDir,
	}

	type mergedEntry struct {
		contents []string
		sources  []SourceLocation
	}
	mergedSchemas := make(map[string]*mergedEntry)
	var mergedSchemasOrder []string

	for _, document := range documents {
		if document.UchiVersion != "v1" {
			continue
		}

		relativePath, err := filepath.Rel(cfg.InputDir, document.FilePath)
		if err != nil || strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
			return nil, &markdown.Diagnostic{
				Path:    document.FilePath,
				Message: fmt.Sprintf("invalid input relative path %q", relativePath),
			}
		}

		ext := filepath.Ext(relativePath)
		relBase := strings.TrimSuffix(relativePath, ext)
		baseStem := filepath.Base(relBase)
		if strings.TrimSpace(relBase) == "" || strings.HasSuffix(relBase, "/") || strings.HasSuffix(relBase, "\\") || baseStem == "." || baseStem == ".." || strings.TrimSpace(baseStem) == "" {
			return nil, &markdown.Diagnostic{
				Path:    document.FilePath,
				Message: fmt.Sprintf("invalid output base name for document %q", document.FilePath),
			}
		}

		docPlan := DocumentPlan{
			SourcePath:   document.FilePath,
			RelativeBase: relBase,
		}

		type schemaGroup struct {
			schema   string
			contents []string
			sources  []SourceLocation
		}
		var docSchemaGroups []schemaGroup
		schemaGroupMap := make(map[string]int)

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

			src := SourceLocation{
				Path: document.FilePath,
				Line: fence.StartLine,
			}

			snippet := ExtractedSnippet{
				Source:           src,
				Language:         fence.Language,
				Schema:           schemaName,
				RawContent:       fence.Content,
				ProcessedContent: processed,
			}
			docPlan.Snippets = append(docPlan.Snippets, snippet)

			idx, exists := schemaGroupMap[schemaName]
			if !exists {
				idx = len(docSchemaGroups)
				schemaGroupMap[schemaName] = idx
				docSchemaGroups = append(docSchemaGroups, schemaGroup{schema: schemaName})
			}
			docSchemaGroups[idx].contents = append(docSchemaGroups[idx].contents, processed)
			docSchemaGroups[idx].sources = append(docSchemaGroups[idx].sources, src)
		}

		if len(docPlan.Snippets) > 0 {
			plan.Documents = append(plan.Documents, docPlan)
		}

		for _, group := range docSchemaGroups {
			outPath, err := buildOutputPath(cfg.OutputDir, "parts", relBase, group.schema)
			if err != nil {
				return nil, &markdown.Diagnostic{
					Path:    document.FilePath,
					Message: fmt.Sprintf("invalid output path for parts: %v", err),
				}
			}

			var nonEmpty []string
			for _, c := range group.contents {
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

			relPartPath := filepath.Join("parts", relBase, group.schema)
			target := TargetFile{
				Path:         outPath,
				RelativePath: relPartPath,
				Schema:       group.schema,
				Kind:         TargetKindPart,
				Content:      data,
				Sources:      group.sources,
			}
			plan.Targets = append(plan.Targets, target)

			entry, exists := mergedSchemas[group.schema]
			if !exists {
				entry = &mergedEntry{}
				mergedSchemas[group.schema] = entry
				mergedSchemasOrder = append(mergedSchemasOrder, group.schema)
			}
			entry.contents = append(entry.contents, fileCombined)
			entry.sources = append(entry.sources, group.sources...)
		}
	}

	for _, schemaName := range mergedSchemasOrder {
		entry := mergedSchemas[schemaName]
		outPath, err := buildOutputPath(cfg.OutputDir, schemaName)
		if err != nil {
			return nil, fmt.Errorf("invalid output path for merged schema %s: %w", schemaName, err)
		}

		var nonEmpty []string
		for _, c := range entry.contents {
			if strings.TrimSpace(c) != "" {
				nonEmpty = append(nonEmpty, c)
			}
		}
		mergedCombined := strings.Join(nonEmpty, "\n\n")
		data := []byte(formatOutput(mergedCombined))

		target := TargetFile{
			Path:         outPath,
			RelativePath: schemaName,
			Schema:       schemaName,
			Kind:         TargetKindMerged,
			Content:      data,
			Sources:      entry.sources,
		}
		plan.Targets = append(plan.Targets, target)
	}

	return plan, nil
}

func verifyGenerationPlan(plan *GenerationPlan) error {
	if plan == nil {
		return fmt.Errorf("generation plan is nil")
	}

	seenPaths := make(map[string]TargetFile)
	for _, target := range plan.Targets {
		if strings.TrimSpace(target.Path) == "" {
			return fmt.Errorf("target file has empty path")
		}

		if existing, exists := seenPaths[target.Path]; exists {
			return fmt.Errorf("conflicting target file path %q (schemas: %s, %s)", target.Path, existing.Schema, target.Schema)
		}
		seenPaths[target.Path] = target
	}

	return nil
}

func executeGenerationPlan(plan *GenerationPlan) error {
	if err := os.MkdirAll(plan.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", plan.OutputDir, err)
	}

	for _, target := range plan.Targets {
		if err := atomicFileWriter(target.Path, target.Content, 0644); err != nil {
			return err
		}
	}

	return nil
}

func runExtraction(cfg *config.Config) error {
	plan, err := buildGenerationPlan(cfg)
	if err != nil {
		return err
	}

	if err := verifyGenerationPlan(plan); err != nil {
		return err
	}

	return executeGenerationPlan(plan)
}

var atomicFileWriter = writeFileAtomic

func writeFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	mode := perm
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}

	pattern := "." + filepath.Base(path) + ".tmp-*"
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return fmt.Errorf("failed to create temp file for %s: %w", path, err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if err := tmpFile.Chmod(mode); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to chmod temp file for %s: %w", path, err)
	}

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file for %s: %w", path, err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file for %s: %w", path, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file for %s: %w", path, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename temp file for %s: %w", path, err)
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
