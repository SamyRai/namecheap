package dns

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"namecheap-dns-manager/internal/testutil"
	"namecheap-dns-manager/pkg/client"
	"namecheap-dns-manager/pkg/config"
)

// ServiceTestSuite is a test suite for DNS service
type ServiceTestSuite struct {
	suite.Suite
	service *Service
}

// TestServiceSuite runs the DNS service test suite
func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupTest() {
	fixture := testutil.AccountConfigFixture()
	accountConfig := &config.AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}
	c, err := client.NewClient(accountConfig)
	s.Require().NoError(err)
	s.service = NewService(c)
}

func (s *ServiceTestSuite) TestService_ValidateRecord_ValidRecords() {
	tests := []struct {
		name   string
		record Record
	}{
		{
			name: "valid A record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeA, "192.168.1.1", 1800, 0)),
		},
		{
			name: "valid AAAA record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", RecordTypeAAAA, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", 1800, 0)),
		},
		{
			name: "valid MX record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeMX, "mail.example.com", 1800, 10)),
		},
		{
			name: "valid CNAME record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", RecordTypeCNAME, "example.com", 1800, 0)),
		},
		{
			name: "valid TXT record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeTXT, "v=spf1 include:_spf.example.com ~all", 1800, 0)),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.service.ValidateRecord(tt.record)
			s.Require().NoError(err)
		})
	}
}

func (s *ServiceTestSuite) TestService_ValidateRecord_InvalidRecords() {
	tests := []struct {
		name   string
		record Record
	}{
		{
			name:   "empty hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("", RecordTypeA, "192.168.1.1", 0, 0)),
		},
		{
			name:   "empty record type",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", "", "192.168.1.1", 0, 0)),
		},
		{
			name:   "empty address",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeA, "", 0, 0)),
		},
		{
			name:   "invalid record type",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", "INVALID", "192.168.1.1", 0, 0)),
		},
		{
			name:   "TTL too low",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeA, "192.168.1.1", 30, 0)),
		},
		{
			name:   "TTL too high",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeA, "192.168.1.1", 100000, 0)),
		},
		{
			name:   "A record with invalid IPv4",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeA, "invalid.ip", 1800, 0)),
		},
		{
			name:   "AAAA record with invalid IPv6",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeAAAA, "192.168.1.1", 1800, 0)),
		},
		{
			name:   "MX record without preference",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeMX, "mail.example.com", 1800, 0)),
		},
		{
			name:   "MX record with invalid hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeMX, "invalid..hostname", 1800, 10)),
		},
		{
			name:   "CNAME record with invalid hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeCNAME, "invalid..hostname", 1800, 0)),
		},
		{
			name:   "NS record with invalid hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeNS, "invalid..hostname", 1800, 0)),
		},
		{
			name:   "MX preference too low",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeMX, "mail.example.com", 1800, -1)),
		},
		{
			name:   "MX preference too high",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", RecordTypeMX, "mail.example.com", 1800, 70000)),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.service.ValidateRecord(tt.record)
			s.Require().Error(err)
		})
	}
}

func (s *ServiceTestSuite) TestService_GetRecordsByType() {
	// Test that the service is created correctly
	s.Require().NotNil(s.service)
	s.Require().NotNil(s.service.client)
}

// convertDNSRecord converts testutil DNS record to dns.Record
func convertDNSRecord(fixture testutil.TestDNSRecord) Record {
	return Record{
		HostName:   fixture.HostName,
		RecordType: fixture.RecordType,
		Address:    fixture.Address,
		TTL:        fixture.TTL,
		MXPref:     fixture.MXPref,
	}
}
