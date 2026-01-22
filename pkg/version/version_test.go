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
	// Test with current version (0.1.0 should be pre-release)
	result := IsPreRelease()
	s.Require().True(result, "Version 0.1.0 should be considered pre-release")
}

func (s *VersionTestSuite) TestIsMajorRelease() {
	// Test with current version (0.1.0 should not be major)
	result := IsMajorRelease()
	s.Require().False(result, "Version 0.1.0 should not be considered major release")
}
