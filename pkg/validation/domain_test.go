package validation

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// DomainValidationTestSuite is a test suite for domain validation
type DomainValidationTestSuite struct {
	suite.Suite
}

// TestDomainValidationSuite runs the domain validation test suite
func TestDomainValidationSuite(t *testing.T) {
	suite.Run(t, new(DomainValidationTestSuite))
}

func (s *DomainValidationTestSuite) TestValidateDomain() {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{
			name:    "valid domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			domain:  "sub.example.com",
			wantErr: false,
		},
		{
			name:    "valid multi-level domain",
			domain:  "www.sub.example.com",
			wantErr: false,
		},
		{
			name:    "empty domain",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "domain without dot",
			domain:  "example",
			wantErr: true,
		},
		{
			name:    "domain too long",
			domain:  string(make([]byte, 254)),
			wantErr: true,
		},
		{
			name:    "domain with empty label",
			domain:  "example..com",
			wantErr: true,
		},
		{
			name:    "domain with label too long",
			domain:  string(make([]byte, 64)) + ".com",
			wantErr: true,
		},
		{
			name:    "valid TLD",
			domain:  "example.co.uk",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateDomain(tt.domain)
			if tt.wantErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
