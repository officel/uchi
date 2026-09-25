package color

import (
	"io"
	"os"
)

const (
	ModeAuto   = "auto"
	ModeAlways = "always"
	ModeNever  = "never"
)

// ValidModes returns the supported color mode strings.
func ValidModes() []string {
	return []string{ModeAuto, ModeAlways, ModeNever}
}

// IsValidMode reports whether mode is a recognized color mode string.
func IsValidMode(mode string) bool {
	switch mode {
	case ModeAuto, ModeAlways, ModeNever:
		return true
	default:
		return false
	}
}

// Colorizer manages formatting strings with ANSI color escape codes.
type Colorizer struct {
	enabled bool
}

// New creates a Colorizer for the specified mode and output writer.
func New(mode string, w io.Writer) *Colorizer {
	return &Colorizer{
		enabled: ShouldColor(mode, w),
	}
}

// ShouldColor determines if colors should be enabled based on mode, NO_COLOR, and writer.
func ShouldColor(mode string, w io.Writer) bool {
	switch mode {
	case ModeNever:
		return false
	case ModeAlways:
		return true
	case ModeAuto, "":
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		if f, ok := w.(*os.File); ok {
			fi, err := f.Stat()
			if err == nil {
				return (fi.Mode() & os.ModeCharDevice) != 0
			}
		}
		return false
	default:
		return false
	}
}

// Enabled reports whether color formatting is active.
func (c *Colorizer) Enabled() bool {
	return c != nil && c.enabled
}

// Red formats text with red ANSI escape codes if color is enabled.
func (c *Colorizer) Red(text string) string {
	if !c.Enabled() || text == "" {
		return text
	}
	return "\033[31m" + text + "\033[0m"
}

// Green formats text with green ANSI escape codes if color is enabled.
func (c *Colorizer) Green(text string) string {
	if !c.Enabled() || text == "" {
		return text
	}
	return "\033[32m" + text + "\033[0m"
}

// Yellow formats text with yellow ANSI escape codes if color is enabled.
func (c *Colorizer) Yellow(text string) string {
	if !c.Enabled() || text == "" {
		return text
	}
	return "\033[33m" + text + "\033[0m"
}

// Cyan formats text with cyan ANSI escape codes if color is enabled.
func (c *Colorizer) Cyan(text string) string {
	if !c.Enabled() || text == "" {
		return text
	}
	return "\033[36m" + text + "\033[0m"
}

// Bold formats text with bold ANSI escape codes if color is enabled.
func (c *Colorizer) Bold(text string) string {
	if !c.Enabled() || text == "" {
		return text
	}
	return "\033[1m" + text + "\033[0m"
}
