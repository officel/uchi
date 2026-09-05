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
	Language      string
	Content       string
	HasAnnotation bool
	Params        map[string]string
}

// Schema returns the schema attribute value if present.
func (c CodeFence) Schema() string {
	if c.Params == nil {
		return ""
	}
	return c.Params["schema"]
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
				fence = parseFenceHeader(trimmed)
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

func parseFenceHeader(trimmedHeader string) CodeFence {
	headerInfo := strings.TrimPrefix(trimmedHeader, "```")
	headerInfo = strings.TrimSpace(headerInfo)

	fence := CodeFence{
		Params: make(map[string]string),
	}

	startIdx := strings.Index(headerInfo, "{")
	endIdx := strings.LastIndex(headerInfo, "}")

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		fence.HasAnnotation = true
		fence.Language = strings.TrimSpace(headerInfo[:startIdx])
		attrStr := headerInfo[startIdx+1 : endIdx]
		for _, part := range strings.Fields(attrStr) {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				fence.Params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			} else if len(kv) == 1 && kv[0] != "" {
				fence.Params[strings.TrimSpace(kv[0])] = "true"
			}
		}
	} else {
		fence.Language = headerInfo
	}

	return fence
}

func isMarkdownFile(path string) bool {
	extension := filepath.Ext(path)
	return extension == ".md" || extension == ".markdown"
}
