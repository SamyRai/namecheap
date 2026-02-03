package errors

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ErrorsTestSuite is a test suite for error types
type ErrorsTestSuite struct {
	suite.Suite
}

// TestErrorsSuite runs the errors test suite
func TestErrorsSuite(t *testing.T) {
	suite.Run(t, new(ErrorsTestSuite))
}

func (s *ErrorsTestSuite) TestErrInvalidInput_WithField() {
	err := NewInvalidInput("username", "cannot be empty")
	s.Require().NotNil(err)
	s.Require().Equal("username", err.Field)
	s.Require().Equal("cannot be empty", err.Message)
	s.Require().Contains(err.Error(), "invalid input username")
	s.Require().Contains(err.Error(), "cannot be empty")
}

func (s *ErrorsTestSuite) TestErrInvalidInput_WithoutField() {
	err := NewInvalidInput("", "invalid configuration")
	s.Require().NotNil(err)
	s.Require().Equal("", err.Field)
	s.Require().Equal("invalid configuration", err.Message)
	s.Require().Contains(err.Error(), "invalid input")
	s.Require().Contains(err.Error(), "invalid configuration")
	s.Require().NotContains(err.Error(), "invalid input :")
}

func (s *ErrorsTestSuite) TestErrNotFound_WithID() {
	err := NewNotFound("DNS record", "example.com")
	s.Require().NotNil(err)
	s.Require().Equal("DNS record", err.Resource)
	s.Require().Equal("example.com", err.ID)
	s.Require().Contains(err.Error(), "DNS record")
	s.Require().Contains(err.Error(), "example.com")
}

func (s *ErrorsTestSuite) TestErrNotFound_WithoutID() {
	err := NewNotFound("DNS record", "")
	s.Require().NotNil(err)
	s.Require().Equal("DNS record", err.Resource)
	s.Require().Equal("", err.ID)
	s.Require().Contains(err.Error(), "DNS record")
	s.Require().Contains(err.Error(), "not found")
	s.Require().NotContains(err.Error(), "'")
}

func (s *ErrorsTestSuite) TestErrConfiguration() {
	err := NewConfiguration("missing required field")
	s.Require().NotNil(err)
	s.Require().Equal("missing required field", err.Message)
	s.Require().Contains(err.Error(), "configuration error")
	s.Require().Contains(err.Error(), "missing required field")
}

func (s *ErrorsTestSuite) TestErrAPI_WithOperation() {
	originalErr := NewConfiguration("underlying error")
	err := NewAPI("GetRecords", "failed to fetch records", originalErr)
	s.Require().NotNil(err)
	s.Require().Equal("GetRecords", err.Operation)
	s.Require().Equal("failed to fetch records", err.Message)
	s.Require().Equal(originalErr, err.Err)
	s.Require().Contains(err.Error(), "API error in GetRecords")
	s.Require().Contains(err.Error(), "failed to fetch records")

	// Test Unwrap
	unwrapped := err.Unwrap()
	s.Require().Equal(originalErr, unwrapped)
}

func (s *ErrorsTestSuite) TestErrAPI_WithoutOperation() {
	originalErr := NewConfiguration("underlying error")
	err := NewAPI("", "failed to fetch records", originalErr)
	s.Require().NotNil(err)
	s.Require().Equal("", err.Operation)
	s.Require().Equal("failed to fetch records", err.Message)
	s.Require().Contains(err.Error(), "API error")
	s.Require().NotContains(err.Error(), "API error in")
}

func (s *ErrorsTestSuite) TestErrAPI_WithoutUnderlyingError() {
	err := NewAPI("GetRecords", "failed to fetch records", nil)
	s.Require().NotNil(err)
	s.Require().Nil(err.Err)
	s.Require().Contains(err.Error(), "API error")

	unwrapped := err.Unwrap()
	s.Require().Nil(unwrapped)
}
