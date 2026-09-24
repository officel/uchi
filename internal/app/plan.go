package app

// SourceLocation represents the origin location of an extracted snippet.
type SourceLocation struct {
	Path string
	Line int
}

// ExtractedSnippet represents a validated code fence and its processed schema content.
type ExtractedSnippet struct {
	Source           SourceLocation
	Language         string
	Schema           string
	RawContent       string
	ProcessedContent string
}

// DocumentPlan represents the extracted intermediate representation of a single Markdown document.
type DocumentPlan struct {
	SourcePath   string
	RelativeBase string
	Snippets     []ExtractedSnippet
}

// TargetKind indicates whether a target file is a part file or a merged schema output.
type TargetKind string

const (
	TargetKindPart   TargetKind = "part"
	TargetKindMerged TargetKind = "merged"
)

// TargetFile represents an individual file to be generated.
type TargetFile struct {
	Path         string
	RelativePath string
	Schema       string
	Kind         TargetKind
	Content      []byte
	Sources      []SourceLocation
}

// GenerationPlan represents the complete set of intermediate representation data
// and target output file specifications needed for generation.
type GenerationPlan struct {
	OutputDir string
	Documents []DocumentPlan
	Targets   []TargetFile
}
