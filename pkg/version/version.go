package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version of the application
	// Set via ldflags during build: -X github.com/concord-chat/concord/pkg/version.Version=1.0.0
	Version = "0.1.0-dev"

	// GitCommit is the git commit hash
	// Set via ldflags during build
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built
	// Set via ldflags during build
	BuildDate = "unknown"

	// GoVersion is the version of Go used to compile
	GoVersion = runtime.Version()

	// Platform is the OS/Arch combination
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf(
		"Concord %s (commit: %s, built: %s, go: %s, platform: %s)",
		i.Version,
		i.GitCommit,
		i.BuildDate,
		i.GoVersion,
		i.Platform,
	)
}

// Short returns a short version string
func (i Info) Short() string {
	return fmt.Sprintf("v%s", i.Version)
}
