// Package version holds build-time version information set via ldflags.
package version

var (
	Version   = "dev"
	Commit    = "unknown"
	Branch    = "unknown"
	BuildTime = "unknown"
	BuildBy   = "unknown"
)
