package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
		return runExtraction(cfg, output)
	case "diff":
		cfg.Diff = true
		return runExtraction(cfg, output)
	case "new":
		return runNew(cfg, output)
	case "init":
		return runInit(cfg, output)
	case "", "check":
		if cfg.Diff {
			return runExtraction(cfg, output)
		}
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

	plan, err := buildGenerationPlan(cfg)
	if err != nil {
		return err
	}

	if err := verifyGenerationPlan(plan); err != nil {
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
				Targets:          fence.Targets,
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

func formatSources(sources []SourceLocation) string {
	if len(sources) == 0 {
		return ""
	}
	var srcStrs []string
	seen := make(map[string]bool)
	for _, s := range sources {
		str := fmt.Sprintf("%s:%d", s.Path, s.Line)
		if !seen[str] {
			seen[str] = true
			srcStrs = append(srcStrs, str)
		}
	}
	return strings.Join(srcStrs, ", ")
}

func verifyGenerationPlan(plan *GenerationPlan) error {
	if plan == nil {
		return fmt.Errorf("generation plan is nil")
	}

	reservedPartsPath := filepath.Clean(filepath.Join(plan.OutputDir, "parts"))
	cleanOutputDir := filepath.Clean(plan.OutputDir)

	pathToTargets := make(map[string][]TargetFile)
	var pathOrder []string

	for _, target := range plan.Targets {
		if strings.TrimSpace(target.Path) == "" {
			return fmt.Errorf("target file has empty path")
		}
		cleanPath := filepath.Clean(target.Path)

		if _, exists := pathToTargets[cleanPath]; !exists {
			pathOrder = append(pathOrder, cleanPath)
		}
		pathToTargets[cleanPath] = append(pathToTargets[cleanPath], target)
	}

	var errs []error

	for _, path := range pathOrder {
		targets := pathToTargets[path]

		if path == reservedPartsPath || path == cleanOutputDir {
			var sources []SourceLocation
			for _, t := range targets {
				sources = append(sources, t.Sources...)
			}
			msg := fmt.Sprintf("target path %q conflicts with reserved directory", path)
			srcsFormatted := formatSources(sources)
			if srcsFormatted != "" {
				msg += fmt.Sprintf(" (sources: %s)", srcsFormatted)
			}
			firstPath := path
			firstLine := 0
			if len(sources) > 0 {
				firstPath = sources[0].Path
				firstLine = sources[0].Line
			}
			errs = append(errs, &markdown.Diagnostic{
				Path:    firstPath,
				Line:    firstLine,
				Message: msg,
			})
			continue
		}

		if len(targets) > 1 {
			var sources []SourceLocation
			var kinds []string
			var schemas []string
			for _, t := range targets {
				sources = append(sources, t.Sources...)
				kinds = append(kinds, string(t.Kind))
				schemas = append(schemas, t.Schema)
			}

			msg := fmt.Sprintf("conflicting target file path %q (kinds: %s; schemas: %s)",
				path, strings.Join(kinds, ", "), strings.Join(schemas, ", "))
			srcsFormatted := formatSources(sources)
			if srcsFormatted != "" {
				msg += fmt.Sprintf(" (sources: %s)", srcsFormatted)
			}

			firstPath := path
			firstLine := 0
			if len(sources) > 0 {
				firstPath = sources[0].Path
				firstLine = sources[0].Line
			}

			errs = append(errs, &markdown.Diagnostic{
				Path:    firstPath,
				Line:    firstLine,
				Message: msg,
			})
		}
	}

	for i := 0; i < len(pathOrder); i++ {
		pathA := pathOrder[i]
		for j := 0; j < len(pathOrder); j++ {
			if i == j {
				continue
			}
			pathB := pathOrder[j]

			rel, err := filepath.Rel(pathA, pathB)
			if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !strings.HasPrefix(rel, "../") && !strings.HasPrefix(rel, "..\\") {
				targetsA := pathToTargets[pathA]
				var sources []SourceLocation
				for _, t := range targetsA {
					sources = append(sources, t.Sources...)
				}
				firstPath := pathA
				firstLine := 0
				if len(sources) > 0 {
					firstPath = sources[0].Path
					firstLine = sources[0].Line
				}
				msg := fmt.Sprintf("target path %q conflicts with target subpath %q", pathA, pathB)
				srcsFormatted := formatSources(sources)
				if srcsFormatted != "" {
					msg += fmt.Sprintf(" (sources: %s)", srcsFormatted)
				}
				errs = append(errs, &markdown.Diagnostic{
					Path:    firstPath,
					Line:    firstLine,
					Message: msg,
				})
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
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

func runExtraction(cfg *config.Config, output io.Writer) error {
	plan, err := buildGenerationPlan(cfg)
	if err != nil {
		return err
	}

	if err := verifyGenerationPlan(plan); err != nil {
		return err
	}

	if cfg.DryRun {
		for _, target := range plan.Targets {
			fmt.Fprintln(output, target.Path)
		}
		return nil
	}

	if cfg.Diff {
		return runDiff(cfg, plan, output)
	}

	if err := executeGenerationPlan(plan); err != nil {
		return err
	}

	return saveManifest(plan)
}

const ManifestFileName = ".uchi-manifest.json"

type Manifest struct {
	Targets []string `json:"targets"`
}

func saveManifest(plan *GenerationPlan) error {
	manifestPath := filepath.Join(plan.OutputDir, ManifestFileName)
	var relPaths []string
	for _, target := range plan.Targets {
		relPaths = append(relPaths, target.RelativePath)
	}
	sort.Strings(relPaths)

	manifest := Manifest{
		Targets: relPaths,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return atomicFileWriter(manifestPath, append(data, '\n'), 0644)
}

func loadManifest(outputDir string) (*Manifest, error) {
	manifestPath := filepath.Join(outputDir, ManifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read manifest %s: %w", manifestPath, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest %s: %w", manifestPath, err)
	}

	return &manifest, nil
}

type DiffOpKind int

const (
	DiffEqual DiffOpKind = iota
	DiffDelete
	DiffInsert
)

type DiffOp struct {
	Kind DiffOpKind
	Line string
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\r\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{""}
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func computeLineDiff(oldLines, newLines []string) []DiffOp {
	m := len(oldLines)
	n := len(newLines)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			if oldLines[i] == newLines[j] {
				dp[i+1][j+1] = dp[i][j] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i+1][j+1] = dp[i+1][j]
			} else {
				dp[i+1][j+1] = dp[i][j+1]
			}
		}
	}

	var ops []DiffOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, DiffOp{Kind: DiffEqual, Line: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, DiffOp{Kind: DiffInsert, Line: newLines[j-1]})
			j--
		} else if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
			ops = append(ops, DiffOp{Kind: DiffDelete, Line: oldLines[i-1]})
			i--
		}
	}

	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	return ops
}

type diffItem struct {
	path   string
	kind   TargetKind
	status string
	symbol string
	ops    []DiffOp
}

func runDiff(cfg *config.Config, plan *GenerationPlan, output io.Writer) error {
	manifest, err := loadManifest(plan.OutputDir)
	if err != nil {
		return err
	}

	var items []diffItem
	planRelPaths := make(map[string]bool)

	for _, target := range plan.Targets {
		planRelPaths[target.RelativePath] = true

		existingData, err := os.ReadFile(target.Path)
		if err != nil {
			if os.IsNotExist(err) {
				lines := splitLines(string(target.Content))
				var ops []DiffOp
				for _, line := range lines {
					ops = append(ops, DiffOp{Kind: DiffInsert, Line: line})
				}
				items = append(items, diffItem{
					path:   target.Path,
					kind:   target.Kind,
					status: "new",
					symbol: "+",
					ops:    ops,
				})
			} else {
				return fmt.Errorf("failed to read target file %s: %w", target.Path, err)
			}
		} else {
			if bytes.Equal(existingData, target.Content) {
				items = append(items, diffItem{
					path:   target.Path,
					kind:   target.Kind,
					status: "unchanged",
					symbol: "=",
				})
			} else {
				oldLines := splitLines(string(existingData))
				newLines := splitLines(string(target.Content))
				ops := computeLineDiff(oldLines, newLines)
				items = append(items, diffItem{
					path:   target.Path,
					kind:   target.Kind,
					status: "modified",
					symbol: "~",
					ops:    ops,
				})
			}
		}
	}

	if manifest != nil {
		var deletedRelPaths []string
		for _, relPath := range manifest.Targets {
			if !planRelPaths[relPath] {
				deletedRelPaths = append(deletedRelPaths, relPath)
			}
		}
		sort.Strings(deletedRelPaths)

		for _, relPath := range deletedRelPaths {
			fullPath := filepath.Join(plan.OutputDir, relPath)
			existingData, err := os.ReadFile(fullPath)
			if err == nil {
				kind := TargetKindMerged
				if strings.HasPrefix(filepath.ToSlash(relPath), "parts/") {
					kind = TargetKindPart
				}
				oldLines := splitLines(string(existingData))
				var ops []DiffOp
				for _, line := range oldLines {
					ops = append(ops, DiffOp{Kind: DiffDelete, Line: line})
				}
				items = append(items, diffItem{
					path:   fullPath,
					kind:   kind,
					status: "deleted",
					symbol: "-",
					ops:    ops,
				})
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to read file %s: %w", fullPath, err)
			}
		}
	}

	for i, item := range items {
		if i > 0 {
			fmt.Fprintln(output)
		}
		fmt.Fprintf(output, "[%s] %s (kind: %s, status: %s)\n", item.symbol, item.path, item.kind, item.status)
		for _, op := range item.ops {
			switch op.Kind {
			case DiffEqual:
				fmt.Fprintf(output, "  %s\n", op.Line)
			case DiffDelete:
				fmt.Fprintf(output, "- %s\n", op.Line)
			case DiffInsert:
				fmt.Fprintf(output, "+ %s\n", op.Line)
			}
		}
	}

	return nil
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
