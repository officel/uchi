package color

import (
	"bytes"
	"testing"
)

func TestValidModes(t *testing.T) {
	modes := ValidModes()
	if len(modes) != 3 {
		t.Fatalf("expected 3 valid modes, got %d", len(modes))
	}
	for _, m := range []string{ModeAuto, ModeAlways, ModeNever} {
		if !IsValidMode(m) {
			t.Errorf("expected IsValidMode(%q) to be true", m)
		}
	}
	if IsValidMode("invalid") {
		t.Errorf("expected IsValidMode(\"invalid\") to be false")
	}
}

func TestShouldColor(t *testing.T) {
	buf := &bytes.Buffer{}

	t.Run("never mode always returns false", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		if ShouldColor(ModeNever, buf) {
			t.Errorf("expected ShouldColor(ModeNever) to be false")
		}
	})

	t.Run("always mode always returns true", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		if !ShouldColor(ModeAlways, buf) {
			t.Errorf("expected ShouldColor(ModeAlways) to be true")
		}
	})

	t.Run("auto mode with NO_COLOR returns false", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		if ShouldColor(ModeAuto, buf) {
			t.Errorf("expected ShouldColor(ModeAuto) with NO_COLOR to be false")
		}
	})

	t.Run("auto mode with bytes buffer returns false", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		if ShouldColor(ModeAuto, buf) {
			t.Errorf("expected ShouldColor(ModeAuto) with bytes.Buffer to be false")
		}
	})
}

func TestColorizerFormatting(t *testing.T) {
	buf := &bytes.Buffer{}

	t.Run("always mode formats with ANSI escape codes", func(t *testing.T) {
		c := New(ModeAlways, buf)
		if !c.Enabled() {
			t.Fatalf("expected c.Enabled() to be true")
		}

		if got := c.Red("error"); got != "\033[31merror\033[0m" {
			t.Errorf("Red() mismatch, got %q", got)
		}
		if got := c.Green("success"); got != "\033[32msuccess\033[0m" {
			t.Errorf("Green() mismatch, got %q", got)
		}
		if got := c.Yellow("warning"); got != "\033[33mwarning\033[0m" {
			t.Errorf("Yellow() mismatch, got %q", got)
		}
		if got := c.Cyan("info"); got != "\033[36minfo\033[0m" {
			t.Errorf("Cyan() mismatch, got %q", got)
		}
		if got := c.Bold("bold"); got != "\033[1mbold\033[0m" {
			t.Errorf("Bold() mismatch, got %q", got)
		}
		if got := c.Red(""); got != "" {
			t.Errorf("Red(\"\") should return empty string, got %q", got)
		}
	})

	t.Run("never mode returns plain text", func(t *testing.T) {
		c := New(ModeNever, buf)
		if c.Enabled() {
			t.Fatalf("expected c.Enabled() to be false")
		}

		if got := c.Red("error"); got != "error" {
			t.Errorf("expected plain text, got %q", got)
		}
		if got := c.Green("success"); got != "success" {
			t.Errorf("expected plain text, got %q", got)
		}
	})

	t.Run("nil colorizer returns plain text", func(t *testing.T) {
		var c *Colorizer
		if c.Enabled() {
			t.Fatalf("expected nil colorizer Enabled() to be false")
		}
		if got := c.Red("error"); got != "error" {
			t.Errorf("expected plain text, got %q", got)
		}
	})
}
