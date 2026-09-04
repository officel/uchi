package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CodeFence represents an extracted code block.
type CodeFence struct {
	Language string
	Content  string
}

// MarkdownDocument represents a parsed markdown document with optional frontmatter and extracted code fences.
type MarkdownDocument struct {
	FilePath    string
	Frontmatter string
	CodeFences  []CodeFence
}

// Run executes the core processing pipeline: scanning input directory for markdown files,
// parsing code fences, and writing output files.
func Run(cfg *Config) error {
	docs, err := ProcessDirectory(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to process directory %s: %w", cfg.InputDir, err)
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", cfg.OutputDir, err)
	}

	for _, doc := range docs {
		if len(doc.CodeFences) == 0 {
			continue
		}

		relPath, err := filepath.Rel(cfg.InputDir, doc.FilePath)
		if err != nil {
			relPath = filepath.Base(doc.FilePath)
		}

		// Save extracted code fences
		ext := filepath.Ext(relPath)
		base := strings.TrimSuffix(relPath, ext)

		for i, fence := range doc.CodeFences {
			var outFilename string
			if len(doc.CodeFences) == 1 {
				outFilename = base + ".txt"
			} else {
				outFilename = fmt.Sprintf("%s_%d.txt", base, i+1)
			}

			outPath := filepath.Join(cfg.OutputDir, outFilename)
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}

			if err := os.WriteFile(outPath, []byte(fence.Content), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", outPath, err)
			}
		}
	}

	return nil
}

// ProcessDirectory walks input directory and processes all .md files.
func ProcessDirectory(dir string) ([]MarkdownDocument, error) {
	var docs []MarkdownDocument

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (filepath.Ext(path) == ".md" || filepath.Ext(path) == ".markdown") {
			doc, err := ParseMarkdownFile(path)
			if err != nil {
				return err
			}
			docs = append(docs, doc)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return docs, nil
}

// ParseMarkdownFile parses frontmatter and extracts code fences from a markdown file.
func ParseMarkdownFile(path string) (MarkdownDocument, error) {
	file, err := os.Open(path)
	if err != nil {
		return MarkdownDocument{}, err
	}
	defer file.Close()

	doc := MarkdownDocument{FilePath: path}
	scanner := bufio.NewScanner(file)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return doc, err
	}

	// Parse Frontmatter if present
	lineIdx := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		var fmLines []string
		lineIdx++
		for lineIdx < len(lines) {
			if strings.TrimSpace(lines[lineIdx]) == "---" {
				lineIdx++
				break
			}
			fmLines = append(fmLines, lines[lineIdx])
			lineIdx++
		}
		doc.Frontmatter = strings.Join(fmLines, "\n")
	}

	// Extract code fences
	var inFence bool
	var currentFence CodeFence
	var fenceLines []string

	for lineIdx < len(lines) {
		line := lines[lineIdx]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				currentFence = CodeFence{
					Language: strings.TrimPrefix(trimmed, "```"),
				}
				fenceLines = nil
			} else {
				inFence = false
				currentFence.Content = strings.Join(fenceLines, "\n")
				doc.CodeFences = append(doc.CodeFences, currentFence)
			}
		} else if inFence {
			fenceLines = append(fenceLines, line)
		}
		lineIdx++
	}

	return doc, nil
}
