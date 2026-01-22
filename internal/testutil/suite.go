package testutil

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// Suite is a base test suite that provides common testing utilities
type Suite struct {
	suite.Suite
}

// NewSuite creates a new test suite instance
func NewSuite(t *testing.T) *Suite {
	s := &Suite{}
	suite.Run(t, s)
	return s
}

// SetupSuite is called once before all tests in the suite
func (s *Suite) SetupSuite() {
	// Override in embedded suites if needed
}

// TearDownSuite is called once after all tests in the suite
func (s *Suite) TearDownSuite() {
	// Override in embedded suites if needed
}

// SetupTest is called before each test
func (s *Suite) SetupTest() {
	// Override in embedded suites if needed
}

// TearDownTest is called after each test
func (s *Suite) TearDownTest() {
	// Override in embedded suites if needed
}

// AssertError checks if an error matches the expected condition
func (s *Suite) AssertError(err error, wantErr bool, msgAndArgs ...interface{}) {
	if wantErr {
		s.Require().Error(err, msgAndArgs...)
	} else {
		s.Require().NoError(err, msgAndArgs...)
	}
}

// AssertNotNil requires that the value is not nil
func (s *Suite) AssertNotNil(value interface{}, msgAndArgs ...interface{}) {
	s.Require().NotNil(value, msgAndArgs...)
}

// AssertNil requires that the value is nil
func (s *Suite) AssertNil(value interface{}, msgAndArgs ...interface{}) {
	s.Require().Nil(value, msgAndArgs...)
}

// AssertEqual requires that two values are equal
func (s *Suite) AssertEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	s.Require().Equal(expected, actual, msgAndArgs...)
}

// AssertNotEqual requires that two values are not equal
func (s *Suite) AssertNotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	s.Require().NotEqual(expected, actual, msgAndArgs...)
}

// AssertTrue requires that the condition is true
func (s *Suite) AssertTrue(condition bool, msgAndArgs ...interface{}) {
	s.Require().True(condition, msgAndArgs...)
}

// AssertFalse requires that the condition is false
func (s *Suite) AssertFalse(condition bool, msgAndArgs ...interface{}) {
	s.Require().False(condition, msgAndArgs...)
}

// AssertContains requires that the container contains the element
func (s *Suite) AssertContains(container, element interface{}, msgAndArgs ...interface{}) {
	s.Require().Contains(container, element, msgAndArgs...)
}

// AssertNotContains requires that the container does not contain the element
func (s *Suite) AssertNotContains(container, element interface{}, msgAndArgs ...interface{}) {
	s.Require().NotContains(container, element, msgAndArgs...)
}

// AssertLen requires that the object has the expected length
func (s *Suite) AssertLen(object interface{}, length int, msgAndArgs ...interface{}) {
	s.Require().Len(object, length, msgAndArgs...)
}

// AssertEmpty requires that the object is empty
func (s *Suite) AssertEmpty(object interface{}, msgAndArgs ...interface{}) {
	s.Require().Empty(object, msgAndArgs...)
}

// AssertNotEmpty requires that the object is not empty
func (s *Suite) AssertNotEmpty(object interface{}, msgAndArgs ...interface{}) {
	s.Require().NotEmpty(object, msgAndArgs...)
}
