package pointer

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// PointerTestSuite is a test suite for pointer utilities
type PointerTestSuite struct {
	suite.Suite
}

// TestPointerSuite runs the pointer test suite
func TestPointerSuite(t *testing.T) {
	suite.Run(t, new(PointerTestSuite))
}

func (s *PointerTestSuite) TestString() {
	tests := []struct {
		name string
		ptr  *string
		want string
	}{
		{
			name: "non-nil string",
			ptr:  stringPtr("test"),
			want: "test",
		},
		{
			name: "nil string",
			ptr:  nil,
			want: "",
		},
		{
			name: "empty string",
			ptr:  stringPtr(""),
			want: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := String(tt.ptr)
			s.AssertEqual(tt.want, got)
		})
	}
}

func (s *PointerTestSuite) TestInt() {
	tests := []struct {
		name string
		ptr  *int
		want int
	}{
		{
			name: "non-nil int",
			ptr:  intPtr(42),
			want: 42,
		},
		{
			name: "nil int",
			ptr:  nil,
			want: 0,
		},
		{
			name: "zero int",
			ptr:  intPtr(0),
			want: 0,
		},
		{
			name: "negative int",
			ptr:  intPtr(-10),
			want: -10,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := Int(tt.ptr)
			s.AssertEqual(tt.want, got)
		})
	}
}

func (s *PointerTestSuite) TestBool() {
	tests := []struct {
		name string
		ptr  *bool
		want bool
	}{
		{
			name: "non-nil true",
			ptr:  boolPtr(true),
			want: true,
		},
		{
			name: "non-nil false",
			ptr:  boolPtr(false),
			want: false,
		},
		{
			name: "nil bool",
			ptr:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := Bool(tt.ptr)
			s.AssertEqual(tt.want, got)
		})
	}
}

// Helper methods for cleaner assertions
func (s *PointerTestSuite) AssertEqual(expected, actual interface{}) {
	s.Require().Equal(expected, actual)
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

