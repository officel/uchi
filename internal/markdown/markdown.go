package markdown

import (
	"bufio"
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

// Document represents a parsed markdown document.
type Document struct {
	FilePath    string
	Frontmatter string
	CodeFences  []CodeFence
}

// Walk parses all Markdown files below dir.
func Walk(dir string) ([]Document, error) {
	var documents []Document

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isMarkdownFile(path) {
			return nil
		}
		document, err := ParseFile(path)
		if err != nil {
			return err
		}
		documents = append(documents, document)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return documents, nil
}

// ParseFile parses frontmatter and extracts code fences from a Markdown file.
func ParseFile(path string) (Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer file.Close()

	document := Document{FilePath: path}
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return document, err
	}

	lineIndex := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		lineIndex++
		var frontmatterLines []string
		for lineIndex < len(lines) {
			if strings.TrimSpace(lines[lineIndex]) == "---" {
				lineIndex++
				break
			}
			frontmatterLines = append(frontmatterLines, lines[lineIndex])
			lineIndex++
		}
		document.Frontmatter = strings.Join(frontmatterLines, "\n")
	}

	var inFence bool
	var fence CodeFence
	var fenceLines []string
	for lineIndex < len(lines) {
		line := lines[lineIndex]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				fence = CodeFence{Language: strings.TrimPrefix(trimmed, "```")}
				fenceLines = nil
			} else {
				inFence = false
				fence.Content = strings.Join(fenceLines, "\n")
				document.CodeFences = append(document.CodeFences, fence)
			}
		} else if inFence {
			fenceLines = append(fenceLines, line)
		}
		lineIndex++
	}

	return document, nil
}

func isMarkdownFile(path string) bool {
	extension := filepath.Ext(path)
	return extension == ".md" || extension == ".markdown"
}
