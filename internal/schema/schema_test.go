package schema

import (
	"testing"
)

func TestValidTargets(t *testing.T) {
	want := []string{"all", "bash", "fish", "powershell", "pwsh", "sh", "zsh"}
	got := ValidTargets()
	if len(got) != len(want) {
		t.Fatalf("ValidTargets() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidTargets()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsValidTarget(t *testing.T) {
	valid := ValidTargets()
	for _, target := range valid {
		if !IsValidTarget(target) {
			t.Errorf("IsValidTarget(%q) = false, want true", target)
		}
	}

	invalid := []string{"", "unknown", "BASH", "zsh1", "cmd"}
	for _, target := range invalid {
		if IsValidTarget(target) {
			t.Errorf("IsValidTarget(%q) = true, want false", target)
		}
	}
}

func TestValidSchemas(t *testing.T) {
	want := []string{"alias", "env", "function", "profile", "rc"}
	got := ValidSchemas()
	if len(got) != len(want) {
		t.Fatalf("ValidSchemas() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidSchemas()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsDefined(t *testing.T) {
	validSchemas := ValidSchemas()
	for _, s := range validSchemas {
		if !IsDefined(s) {
			t.Errorf("IsDefined(%q) = false, want true", s)
		}
	}

	invalidSchemas := []string{"", "unknown", "ALIAS", "shell", "bash"}
	for _, s := range invalidSchemas {
		if IsDefined(s) {
			t.Errorf("IsDefined(%q) = true, want false", s)
		}
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		schema string
		input  string
		want   string
		wantOk bool
	}{
		{"alias", "alias ll='ls -la'", "alias ll='ls -la'", true},
		{"env", "export FOO=bar", "export FOO=bar", true},
		{"profile", "umask 022", "umask 022", true},
		{"rc", "set -o vi", "set -o vi", true},
		{"function", "foo() { echo bar; }", "foo() { echo bar; }", true},
		{"invalid", "echo test", "", false},
	}

	for _, tt := range tests {
		got, ok := Process(tt.schema, tt.input)
		if ok != tt.wantOk {
			t.Errorf("Process(%q, %q) ok = %v, want %v", tt.schema, tt.input, ok, tt.wantOk)
		}
		if got != tt.want {
			t.Errorf("Process(%q, %q) = %q, want %q", tt.schema, tt.input, got, tt.want)
		}
	}
}

func TestIndividualFunctions(t *testing.T) {
	if got := ProcessAlias("content"); got != "content" {
		t.Errorf("ProcessAlias() = %q, want %q", got, "content")
	}
	if got := ProcessEnv("content"); got != "content" {
		t.Errorf("ProcessEnv() = %q, want %q", got, "content")
	}
	if got := ProcessProfile("content"); got != "content" {
		t.Errorf("ProcessProfile() = %q, want %q", got, "content")
	}
	if got := ProcessRc("content"); got != "content" {
		t.Errorf("ProcessRc() = %q, want %q", got, "content")
	}
	if got := ProcessFunction("content"); got != "content" {
		t.Errorf("ProcessFunction() = %q, want %q", got, "content")
	}
}
