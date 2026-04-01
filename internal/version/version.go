// Package version holds build-time version information set via ldflags.
package version

// Version is the semantic version or git describe output.
var Version = "dev"

// Commit is the short git commit hash.
var Commit = "unknown"

// Branch is the git branch name.
var Branch = "unknown"

// BuildTime is the UTC timestamp of the build.
var BuildTime = "unknown"

// BuildBy is the user or system that produced the build.
var BuildBy = "unknown"
