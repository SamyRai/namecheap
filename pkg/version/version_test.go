package version

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// VersionTestSuite is a test suite for version package
type VersionTestSuite struct {
	suite.Suite
}

// TestVersionSuite runs the version test suite
func TestVersionSuite(t *testing.T) {
	suite.Run(t, new(VersionTestSuite))
}

func (s *VersionTestSuite) TestGet() {
	info := Get()
	s.Require().NotEmpty(info.Version)
	s.Require().NotEmpty(info.GoVersion)
}

func (s *VersionTestSuite) TestString() {
	versionStr := String()
	s.Require().NotEmpty(versionStr)
	s.Require().Contains(versionStr, Version)
}

func (s *VersionTestSuite) TestFullString() {
	fullStr := FullString()
	s.Require().NotEmpty(fullStr)
	s.Require().Contains(fullStr, "Version:")
	s.Require().Contains(fullStr, "Go Version:")
}

func (s *VersionTestSuite) TestIsPreRelease() {
	// IsPreRelease is a pure semver check; exercise it against a fixed
	// version rather than the package default, which is "dev" unless
	// overridden by -ldflags at build time (O7).
	original := Version
	defer func() { Version = original }()

	Version = "0.1.0"
	s.Require().True(IsPreRelease(), "Version 0.1.0 should be considered pre-release")
}

func (s *VersionTestSuite) TestIsMajorRelease() {
	original := Version
	defer func() { Version = original }()

	Version = "0.1.0"
	s.Require().False(IsMajorRelease(), "Version 0.1.0 should not be considered major release")

	Version = "1.2.0"
	s.Require().True(IsMajorRelease(), "Version 1.2.0 should be considered major release")
}

func (s *VersionTestSuite) TestDefaultVersionIsDev() {
	// Unset by -ldflags, Version must read "dev" rather than a stale
	// hardcoded semver (O7) so a `go build`/`go run` without the release
	// build flags is never mistaken for a tagged release.
	s.Require().Equal("dev", Version)
}

func (s *VersionTestSuite) TestStringIncludesCommitWhenSet() {
	originalCommit := GitCommit
	originalVersion := Version
	defer func() {
		GitCommit = originalCommit
		Version = originalVersion
	}()

	Version = "1.2.3"
	GitCommit = "abcdef1234567890"

	s.Require().Equal("1.2.3 (commit: abcdef1)", String())
}
