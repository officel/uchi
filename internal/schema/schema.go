package schema

// Defined schema constants.
const (
	SchemaAlias    = "alias"
	SchemaEnv      = "env"
	SchemaProfile  = "profile"
	SchemaRc       = "rc"
	SchemaFunction = "function"
)

// Defined target shell constants.
const (
	TargetAll        = "all"
	TargetBash       = "bash"
	TargetFish       = "fish"
	TargetPowerShell = "powershell"
	TargetPwsh       = "pwsh"
	TargetSh         = "sh"
	TargetZsh        = "zsh"
)

var definedTargets = map[string]bool{
	TargetAll:        true,
	TargetBash:       true,
	TargetFish:       true,
	TargetPowerShell: true,
	TargetPwsh:       true,
	TargetSh:         true,
	TargetZsh:        true,
}

// ValidTargets returns a sorted slice of valid target shell names.
func ValidTargets() []string {
	return []string{TargetAll, TargetBash, TargetFish, TargetPowerShell, TargetPwsh, TargetSh, TargetZsh}
}

// IsValidTarget reports whether the given name is a valid target shell name.
func IsValidTarget(name string) bool {
	return definedTargets[name]
}

var definedSchemas = map[string]func(string) string{
	SchemaAlias:    ProcessAlias,
	SchemaEnv:      ProcessEnv,
	SchemaProfile:  ProcessProfile,
	SchemaRc:       ProcessRc,
	SchemaFunction: ProcessFunction,
}

// ValidSchemas returns a sorted slice of defined schema names.
func ValidSchemas() []string {
	return []string{SchemaAlias, SchemaEnv, SchemaFunction, SchemaProfile, SchemaRc}
}

// IsDefined reports whether the given schema name is defined.
func IsDefined(name string) bool {
	_, ok := definedSchemas[name]
	return ok
}

// Process processes content using the schema-specific function for name.
// It returns the processed string and true if defined, or ("", false) if undefined.
func Process(name string, content string) (string, bool) {
	fn, ok := definedSchemas[name]
	if !ok {
		return "", false
	}
	return fn(content), true
}

// ProcessAlias processes code fences with schema=alias.
func ProcessAlias(content string) string {
	return content
}

// ProcessEnv processes code fences with schema=env.
func ProcessEnv(content string) string {
	return content
}

// ProcessProfile processes code fences with schema=profile.
func ProcessProfile(content string) string {
	return content
}

// ProcessRc processes code fences with schema=rc.
func ProcessRc(content string) string {
	return content
}

// ProcessFunction processes code fences with schema=function.
func ProcessFunction(content string) string {
	return content
}
