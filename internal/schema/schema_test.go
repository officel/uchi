package schema

import (
	"testing"
)

func TestIsDefined(t *testing.T) {
	validSchemas := []string{"alias", "env", "profile", "rc", "function"}
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
		schema   string
		input    string
		want     string
		wantOk   bool
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
