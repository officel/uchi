package markdown

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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

// Frontmatter represents parsed frontmatter metadata.
type Frontmatter struct {
	UchiVersion string
}

// ParseFrontmatter parses raw frontmatter YAML and returns structured metadata.
func ParseFrontmatter(raw string) (Frontmatter, error) {
	if strings.TrimSpace(raw) == "" {
		return Frontmatter{}, nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &node); err != nil {
		return Frontmatter{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if len(node.Content) == 0 {
		return Frontmatter{}, nil
	}

	docNode := node.Content[0]
	if docNode.Kind != yaml.MappingNode {
		return Frontmatter{}, nil
	}

	for i := 0; i < len(docNode.Content); i += 2 {
		keyNode := docNode.Content[i]
		valNode := docNode.Content[i+1]

		if keyNode.Value == "uchi" {
			if valNode.Kind != yaml.ScalarNode || (valNode.ShortTag() != "!!str" && valNode.Tag != "") {
				return Frontmatter{}, fmt.Errorf("'uchi' field must be a string")
			}
			if valNode.Value != "v1" {
				return Frontmatter{}, fmt.Errorf("unsupported uchi version %q", valNode.Value)
			}
			return Frontmatter{UchiVersion: valNode.Value}, nil
		}
	}

	return Frontmatter{}, nil
}

// Document represents a parsed markdown document.
type Document struct {
	FilePath    string
	Frontmatter string
	UchiVersion string
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
	hasFrontmatter := false
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		lineIndex++
		hasFrontmatter = true
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

	if hasFrontmatter {
		fm, err := ParseFrontmatter(document.Frontmatter)
		if err != nil {
			return document, fmt.Errorf("failed to parse frontmatter in %s: %w", path, err)
		}
		document.UchiVersion = fm.UchiVersion
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
