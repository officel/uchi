package markdown

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Diagnostic represents a testable diagnostic error containing file path, line number, and cause.
type Diagnostic struct {
	Path    string
	Line    int
	Message string
}

func (d *Diagnostic) Error() string {
	if d.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", d.Path, d.Line, d.Message)
	}
	if d.Path != "" {
		return fmt.Sprintf("%s: %s", d.Path, d.Message)
	}
	return d.Message
}

// CodeFence represents an extracted code block.
type CodeFence struct {
	Language      string
	Content       string
	HasAnnotation bool
	Params        map[string]string
	StartLine     int
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
	FilePath             string
	Frontmatter          string
	UchiVersion          string
	CodeFences           []CodeFence
	FrontmatterStartLine int
}

// Walk parses all Markdown files below dir.
func Walk(dir string) ([]Document, error) {
	var documents []Document
	var parseErrs []error

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isMarkdownFile(path) {
			return nil
		}
		document, err := ParseFile(path)
		if err != nil {
			parseErrs = append(parseErrs, err)
			return nil
		}
		documents = append(documents, document)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	return documents, nil
}

// ParseFile parses frontmatter and extracts code fences from a Markdown file.
func ParseFile(path string) (Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return Document{}, &Diagnostic{
			Path:    path,
			Line:    0,
			Message: fmt.Sprintf("failed to open file: %v", err),
		}
	}
	defer file.Close()

	document := Document{FilePath: path}
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return document, &Diagnostic{
			Path:    path,
			Line:    0,
			Message: fmt.Sprintf("failed to read file: %v", err),
		}
	}

	lineIndex := 0
	hasFrontmatter := false
	fmStartLine := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		fmStartLine = 1
		document.FrontmatterStartLine = fmStartLine
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

	var parseErrs []error

	if hasFrontmatter {
		fm, err := ParseFrontmatter(document.Frontmatter)
		if err != nil {
			parseErrs = append(parseErrs, &Diagnostic{
				Path:    path,
				Line:    fmStartLine,
				Message: fmt.Sprintf("failed to parse frontmatter: %v", err),
			})
		} else {
			document.UchiVersion = fm.UchiVersion
		}
	}

	var inFence bool
	var fence CodeFence
	var fenceLines []string
	var fenceStartLine int
	for lineIndex < len(lines) {
		line := lines[lineIndex]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				fenceStartLine = lineIndex + 1
				var parseErr error
				fence, parseErr = ParseFenceHeader(trimmed)
				if parseErr != nil && document.UchiVersion == "v1" {
					parseErrs = append(parseErrs, &Diagnostic{
						Path:    path,
						Line:    fenceStartLine,
						Message: fmt.Sprintf("failed to parse code fence: %v", parseErr),
					})
				}
				fence.StartLine = fenceStartLine
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

	if len(parseErrs) > 0 {
		return document, errors.Join(parseErrs...)
	}

	return document, nil
}

// ParseFenceHeader parses a code fence header line and returns the extracted CodeFence.
func ParseFenceHeader(trimmedHeader string) (CodeFence, error) {
	headerInfo := strings.TrimPrefix(trimmedHeader, "```")
	headerInfo = strings.TrimSpace(headerInfo)

	fence := CodeFence{
		Params: make(map[string]string),
	}

	startIdx := strings.Index(headerInfo, "{")
	if startIdx != -1 {
		fence.HasAnnotation = true
		fence.Language = strings.TrimSpace(headerInfo[:startIdx])
		if strings.Contains(fence.Language, "}") {
			return fence, fmt.Errorf("unmatched '}' in code fence header")
		}
		params, err := parseAttributeBlock(headerInfo[startIdx:])
		if err != nil {
			return fence, err
		}
		fence.Params = params
	} else {
		if strings.Contains(headerInfo, "}") {
			return fence, fmt.Errorf("unmatched '}' in code fence header")
		}
		fence.Language = headerInfo
	}

	return fence, nil
}

func parseAttributeBlock(attrInput string) (map[string]string, error) {
	if len(attrInput) == 0 || attrInput[0] != '{' {
		return nil, fmt.Errorf("attribute block must start with '{'")
	}

	params := make(map[string]string)
	i := 1 // skip opening '{'
	n := len(attrInput)

	for i < n {
		for i < n && isSpace(attrInput[i]) {
			i++
		}
		if i >= n {
			return nil, fmt.Errorf("unclosed '{' in code fence header")
		}

		if attrInput[i] == '}' {
			remainder := strings.TrimSpace(attrInput[i+1:])
			if remainder != "" {
				return nil, fmt.Errorf("unexpected content %q after '}' in code fence header", remainder)
			}
			return params, nil
		}

		if attrInput[i] == '{' {
			return nil, fmt.Errorf("unexpected '{' inside attribute block")
		}

		keyStart := i
		for i < n && isKeyChar(attrInput[i]) {
			i++
		}
		key := attrInput[keyStart:i]
		if key == "" {
			return nil, fmt.Errorf("invalid or empty attribute name")
		}

		if _, exists := params[key]; exists {
			return nil, fmt.Errorf("duplicate attribute key %q", key)
		}

		for i < n && isSpace(attrInput[i]) {
			i++
		}

		if i >= n {
			return nil, fmt.Errorf("unclosed '{' in code fence header")
		}

		if attrInput[i] == '=' {
			i++ // skip '='
			for i < n && isSpace(attrInput[i]) {
				i++
			}
			if i >= n {
				return nil, fmt.Errorf("unclosed '{' in code fence header")
			}
			if attrInput[i] == '}' {
				return nil, fmt.Errorf("empty attribute value for key %q", key)
			}

			var val string
			if attrInput[i] == '"' {
				i++ // skip opening '"'
				escaped := false
				var sb strings.Builder
				foundClose := false
				for i < n {
					ch := attrInput[i]
					if escaped {
						sb.WriteByte(ch)
						escaped = false
						i++
						continue
					}
					if ch == '\\' {
						escaped = true
						i++
						continue
					}
					if ch == '"' {
						foundClose = true
						i++ // skip closing '"'
						break
					}
					sb.WriteByte(ch)
					i++
				}
				if !foundClose {
					return nil, fmt.Errorf("unclosed double quote for attribute %q", key)
				}
				val = sb.String()
			} else if attrInput[i] == '\'' {
				i++ // skip opening '\''
				escaped := false
				var sb strings.Builder
				foundClose := false
				for i < n {
					ch := attrInput[i]
					if escaped {
						sb.WriteByte(ch)
						escaped = false
						i++
						continue
					}
					if ch == '\\' {
						escaped = true
						i++
						continue
					}
					if ch == '\'' {
						foundClose = true
						i++ // skip closing '\''
						break
					}
					sb.WriteByte(ch)
					i++
				}
				if !foundClose {
					return nil, fmt.Errorf("unclosed single quote for attribute %q", key)
				}
				val = sb.String()
			} else {
				valStart := i
				for i < n && !isSpace(attrInput[i]) && attrInput[i] != '}' && attrInput[i] != '{' && attrInput[i] != '"' && attrInput[i] != '\'' {
					i++
				}
				val = attrInput[valStart:i]
			}

			if val == "" {
				return nil, fmt.Errorf("empty attribute value for key %q", key)
			}

			if i < n && attrInput[i] != '}' && !isSpace(attrInput[i]) {
				return nil, fmt.Errorf("expected whitespace or '}' after value for key %q", key)
			}

			params[key] = val
		} else if attrInput[i] == '}' || isKeyChar(attrInput[i]) {
			params[key] = "true"
		} else {
			return nil, fmt.Errorf("invalid character %q after attribute name %q", attrInput[i], key)
		}
	}

	return nil, fmt.Errorf("unclosed '{' in code fence header")
}

func isKeyChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

func isSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isMarkdownFile(path string) bool {
	extension := filepath.Ext(path)
	return extension == ".md" || extension == ".markdown"
}
