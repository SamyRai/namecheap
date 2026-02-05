package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the application version (semantic versioning)
	// This should be updated when creating releases
	Version = "2.0.0"

	// BuildDate is the build date (set during build)
	BuildDate = "unknown"

	// GitCommit is the git commit hash (set during build)
	GitCommit = "unknown"

	// GitTag is the git tag (set during build)
	GitTag = ""

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// Info holds version information
type Info struct {
	Version   string
	BuildDate string
	GitCommit string
	GitTag    string
	GoVersion string
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GitTag:    GitTag,
		GoVersion: GoVersion,
	}
}

// String returns a formatted version string
func String() string {
	info := Get()
	version := info.Version

	if info.GitTag != "" && info.GitTag != info.Version {
		version = fmt.Sprintf("%s (tag: %s)", info.Version, info.GitTag)
	}

	if info.GitCommit != "unknown" && len(info.GitCommit) > 7 {
		version = fmt.Sprintf("%s (commit: %s)", version, info.GitCommit[:7])
	}

	return version
}

// FullString returns a detailed version string
func FullString() string {
	info := Get()
	return fmt.Sprintf(`Version: %s
Build Date: %s
Git Commit: %s
Git Tag: %s
Go Version: %s
`, info.Version, info.BuildDate, info.GitCommit, info.GitTag, info.GoVersion)
}

// IsPreRelease returns true if the version is a pre-release (alpha, beta, rc)
func IsPreRelease() bool {
	// Check if version contains pre-release identifiers
	// Semantic versioning: 1.0.0-alpha.1, 1.0.0-beta.1, 1.0.0-rc.1
	return len(Version) > 0 && (Version[0] == '0' ||
		contains(Version, "-alpha") ||
		contains(Version, "-beta") ||
		contains(Version, "-rc"))
}

// IsMajorRelease returns true if version is >= 1.0.0
func IsMajorRelease() bool {
	return len(Version) > 0 && Version[0] != '0'
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
