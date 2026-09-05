package template

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
)

//go:embed defaults/*.tmpl
var defaultFiles embed.FS

// Data is available to every rendered template.
type Data struct {
	Name string
	Date string
}

// Resolver loads templates from an optional override directory or bundled defaults.
type Resolver struct {
	directory string
}

// NewResolver returns a resolver that uses directory for template overrides.
func NewResolver(directory string) Resolver {
	return Resolver{directory: directory}
}

// Render loads and renders the named template with data.
func (resolver Resolver) Render(name string, data Data) (string, error) {
	content, err := resolver.load(name)
	if err != nil {
		return "", err
	}

	template, err := texttemplate.New(name).Option("missingkey=error").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var output strings.Builder
	if err := template.Execute(&output, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return output.String(), nil
}

func (resolver Resolver) load(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}

	if resolver.directory != "" {
		path := filepath.Join(resolver.directory, name+".tmpl")
		content, err := os.ReadFile(path)
		switch {
		case err == nil:
			return string(content), nil
		case !os.IsNotExist(err):
			return "", fmt.Errorf("read template override %s: %w", path, err)
		}
	}

	content, err := fs.ReadFile(defaultFiles, filepath.Join("defaults", name+".tmpl"))
	if err != nil {
		return "", fmt.Errorf("template %q is not available: %w", name, err)
	}
	return string(content), nil
}

func validateName(name string) error {
	if name == "" || filepath.Base(name) != name || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid template name %q", name)
	}
	return nil
}
