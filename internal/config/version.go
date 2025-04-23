package config

// Version information set by goreleaser
var (
	// Version holds the current version number
	Version = "dev"

	// Commit holds the git commit hash
	Commit = "none"

	// Date holds the build date
	Date = "unknown"
)

// GetVersionInfo returns a formatted string containing version information
func GetVersionInfo() string {
	return Version + " (" + Commit + ") built at " + Date
}
