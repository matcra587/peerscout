package output

// FormatType represents the output format to use.
type FormatType string

const (
	// FormatAgentJSON outputs JSON wrapped in an agent envelope.
	FormatAgentJSON FormatType = "agent-json"
	// FormatJSON outputs syntax-highlighted JSON on a TTY, plain JSON otherwise.
	FormatJSON FormatType = "json"
	// FormatPlain outputs one item per line.
	FormatPlain FormatType = "plain"
	// FormatCSV outputs comma-separated values on a single line.
	FormatCSV FormatType = "csv"
)

// FormatOpts carries the inputs used to determine output format.
type FormatOpts struct {
	AgentMode bool
	Format    string
}

// DetectFormat returns the appropriate FormatType given opts.
// Priority: agent mode > explicit format flag > plain.
func DetectFormat(opts FormatOpts) FormatType {
	if opts.AgentMode {
		return FormatAgentJSON
	}

	switch opts.Format {
	case "json":
		return FormatJSON
	case "csv":
		return FormatCSV
	}

	return FormatPlain
}
