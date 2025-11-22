package dns

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ValidationTestSuite is a test suite for DNS validation
type ValidationTestSuite struct {
	suite.Suite
}

// TestValidationSuite runs the DNS validation test suite
func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}

func (s *ValidationTestSuite) TestValidateDomain() {
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
			name:    "invalid domain",
			domain:  "",
			wantErr: true,
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

func (s *ValidationTestSuite) TestValidateHostname() {
	tests := []struct {
		name     string
		hostname string
		wantErr  bool
	}{
		{
			name:     "valid hostname",
			hostname: "www.example.com",
			wantErr:  false,
		},
		{
			name:     "root domain marker",
			hostname: "@",
			wantErr:  false,
		},
		{
			name:     "empty hostname",
			hostname: "",
			wantErr:  true,
		},
		{
			name:     "hostname too long",
			hostname: string(make([]byte, 254)),
			wantErr:  true,
		},
		{
			name:     "hostname with empty label",
			hostname: "www..example.com",
			wantErr:  true,
		},
		{
			name:     "hostname with label too long",
			hostname: string(make([]byte, 64)) + ".com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateHostname(tt.hostname)
			if tt.wantErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ValidationTestSuite) TestValidateIPv4() {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "valid IPv4",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid IPv4 localhost",
			ip:      "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "invalid IPv4",
			ip:      "256.256.256.256",
			wantErr: true,
		},
		{
			name:    "IPv6 address",
			ip:      "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr: true,
		},
		{
			name:    "invalid format",
			ip:      "not.an.ip",
			wantErr: true,
		},
		{
			name:    "empty string",
			ip:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateIPv4(tt.ip)
			if tt.wantErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ValidationTestSuite) TestValidateIPv6() {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "valid IPv6",
			ip:      "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr: false,
		},
		{
			name:    "valid IPv6 shortened",
			ip:      "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "IPv4 address",
			ip:      "192.168.1.1",
			wantErr: true,
		},
		{
			name:    "invalid format",
			ip:      "not.an.ip",
			wantErr: true,
		},
		{
			name:    "empty string",
			ip:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateIPv6(tt.ip)
			if tt.wantErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
