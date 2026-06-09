package version

// Set via -ldflags at build time; fallback for development.
var (
	ServerVersion = "0.1.0"
	GitCommit     = "unknown"
)
