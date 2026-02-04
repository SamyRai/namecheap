package dns

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"zonekit/internal/testutil"
	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

// mockProvider is a mock implementation of the Provider interface for testing
type mockProvider struct {
	name            string
	records         map[string][]dnsrecord.Record
	getRecordsError error
	setRecordsError error
	validateError   error
	isAtomic        bool
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:     name,
		records:  make(map[string][]dnsrecord.Record),
		isAtomic: true,
	}
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		IsBulkReplaceAtomic: m.isAtomic,
	}
}

func (m *mockProvider) GetRecords(ctx context.Context, domainName string) ([]dnsrecord.Record, error) {
	if m.getRecordsError != nil {
		return nil, m.getRecordsError
	}
	return m.records[domainName], nil
}

func (m *mockProvider) SetRecords(ctx context.Context, domainName string, records []dnsrecord.Record) error {
	if m.setRecordsError != nil {
		return m.setRecordsError
	}
	m.records[domainName] = records
	return nil
}

func (m *mockProvider) AddRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	m.records[domainName] = append(m.records[domainName], record)
	return nil
}

func (m *mockProvider) UpdateRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	records := m.records[domainName]
	found := false
	for i, r := range records {
		if r.HostName == record.HostName && r.RecordType == record.RecordType {
			records[i] = record
			found = true
			break
		}
	}
	if !found {
		return errors.New("record not found")
	}
	m.records[domainName] = records
	return nil
}

func (m *mockProvider) DeleteRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	records := m.records[domainName]
	var newRecords []dnsrecord.Record
	found := false
	for _, r := range records {
		if r.HostName == record.HostName && r.RecordType == record.RecordType {
			found = true
			continue
		}
		newRecords = append(newRecords, r)
	}
	if !found {
		return errors.New("record not found")
	}
	m.records[domainName] = newRecords
	return nil
}

func (m *mockProvider) Validate() error {
	return m.validateError
}

// ServiceTestSuite is a test suite for DNS service
type ServiceTestSuite struct {
	suite.Suite
	service *Service
	mock    *mockProvider
	ctx     context.Context
}

// TestServiceSuite runs the DNS service test suite
func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupTest() {
	s.mock = newMockProvider("mock")
	s.service = NewServiceWithProvider(s.mock)
	s.ctx = context.Background()
}

func (s *ServiceTestSuite) TestService_ValidateRecord_ValidRecords() {
	tests := []struct {
		name   string
		record dnsrecord.Record
	}{
		{
			name:   "valid A record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		},
		{
			name:   "valid AAAA record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeAAAA, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", 1800, 0)),
		},
		{
			name:   "valid MX record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "mail.example.com", 1800, 10)),
		},
		{
			name:   "valid CNAME record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeCNAME, "example.com", 1800, 0)),
		},
		{
			name:   "valid TXT record",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeTXT, "v=spf1 include:_spf.example.com ~all", 1800, 0)),
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
		record dnsrecord.Record
	}{
		{
			name:   "empty hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("", dnsrecord.RecordTypeA, "192.168.1.1", 0, 0)),
		},
		{
			name:   "empty record type",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", "", "192.168.1.1", 0, 0)),
		},
		{
			name:   "empty address",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "", 0, 0)),
		},
		{
			name:   "invalid record type",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", "INVALID", "192.168.1.1", 0, 0)),
		},
		{
			name:   "TTL too low",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 30, 0)),
		},
		{
			name:   "TTL too high",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 100000, 0)),
		},
		{
			name:   "A record with invalid IPv4",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "invalid.ip", 1800, 0)),
		},
		{
			name:   "AAAA record with invalid IPv6",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeAAAA, "192.168.1.1", 1800, 0)),
		},
		{
			name:   "MX record without preference",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "mail.example.com", 1800, 0)),
		},
		{
			name:   "MX record with invalid hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "invalid..hostname", 1800, 10)),
		},
		{
			name:   "CNAME record with invalid hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeCNAME, "invalid..hostname", 1800, 0)),
		},
		{
			name:   "NS record with invalid hostname",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeNS, "invalid..hostname", 1800, 0)),
		},
		{
			name:   "MX preference too low",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "mail.example.com", 1800, -1)),
		},
		{
			name:   "MX preference too high",
			record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "mail.example.com", 1800, 70000)),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.service.ValidateRecord(tt.record)
			s.Require().Error(err)
		})
	}
}

func (s *ServiceTestSuite) TestService_GetRecords() {
	domain := testutil.ValidDomainFixture()

	// Setup mock records
	mockRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
	}
	s.mock.records[domain] = mockRecords

	// Test GetRecords
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 2)
	s.Require().Equal(mockRecords, records)
}

func (s *ServiceTestSuite) TestService_GetRecords_Error() {
	domain := testutil.ValidDomainFixture()
	expectedError := errors.New("provider error")
	s.mock.getRecordsError = expectedError

	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().Error(err)
	s.Require().Equal(expectedError, err)
	s.Require().Nil(records)
}

func (s *ServiceTestSuite) TestService_SetRecords() {
	domain := testutil.ValidDomainFixture()
	records := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}

	err := s.service.SetRecords(s.ctx, domain, records)
	s.Require().NoError(err)
	s.Require().Equal(records, s.mock.records[domain])
}

func (s *ServiceTestSuite) TestService_SetRecords_Error() {
	domain := testutil.ValidDomainFixture()
	expectedError := errors.New("provider error")
	s.mock.setRecordsError = expectedError

	records := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}

	err := s.service.SetRecords(s.ctx, domain, records)
	s.Require().Error(err)
	s.Require().Equal(expectedError, err)
}

func (s *ServiceTestSuite) TestService_AddRecord() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Add new record
	newRecord := convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0))
	err := s.service.AddRecord(s.ctx, domain, newRecord)
	s.Require().NoError(err)

	// Verify record was added
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	// With Atomic=true (default mock), Service calls SetRecords.
	// We expect 2 records now.
	s.Require().Len(records, 2)
	s.Require().Contains(records, newRecord)
}

func (s *ServiceTestSuite) TestService_UpdateRecord() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Update record
	updatedRecord := convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.100", 3600, 0))
	err := s.service.UpdateRecord(s.ctx, domain, "@", dnsrecord.RecordTypeA, updatedRecord)
	s.Require().NoError(err)

	// Verify record was updated
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal(updatedRecord.Address, records[0].Address)
}

func (s *ServiceTestSuite) TestService_UpdateRecord_Rename() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("old", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Rename record
	updatedRecord := convertDNSRecord(testutil.DNSRecordFixtureWithValues("new", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0))
	err := s.service.UpdateRecord(s.ctx, domain, "old", dnsrecord.RecordTypeA, updatedRecord)
	s.Require().NoError(err)

	// Verify record was updated/renamed
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal("new", records[0].HostName)
}

func (s *ServiceTestSuite) TestService_UpdateRecord_NotFound() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Try to update non-existent record
	updatedRecord := convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0))
	err := s.service.UpdateRecord(s.ctx, domain, "www", dnsrecord.RecordTypeA, updatedRecord)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestService_DeleteRecord() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Delete record
	err := s.service.DeleteRecord(s.ctx, domain, "@", dnsrecord.RecordTypeA)
	s.Require().NoError(err)

	// Verify record was deleted
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal("www", records[0].HostName)
}

func (s *ServiceTestSuite) TestService_DeleteRecord_NotFound() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Try to delete non-existent record
	err := s.service.DeleteRecord(s.ctx, domain, "www", dnsrecord.RecordTypeA)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestService_DeleteAllRecords() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Delete all records
	err := s.service.DeleteAllRecords(s.ctx, domain)
	s.Require().NoError(err)

	// Verify all records were deleted
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Empty(records)
}

func (s *ServiceTestSuite) TestService_GetRecordsByType() {
	domain := testutil.ValidDomainFixture()

	// Setup mock records
	mockRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "mail.example.com", 1800, 10)),
	}
	s.mock.records[domain] = mockRecords

	// Test GetRecordsByType
	aRecords, err := s.service.GetRecordsByType(s.ctx, domain, dnsrecord.RecordTypeA)
	s.Require().NoError(err)
	s.Require().Len(aRecords, 2)
	s.Require().Equal(dnsrecord.RecordTypeA, aRecords[0].RecordType)
	s.Require().Equal(dnsrecord.RecordTypeA, aRecords[1].RecordType)

	mxRecords, err := s.service.GetRecordsByType(s.ctx, domain, dnsrecord.RecordTypeMX)
	s.Require().NoError(err)
	s.Require().Len(mxRecords, 1)
	s.Require().Equal(dnsrecord.RecordTypeMX, mxRecords[0].RecordType)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_Add() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Bulk add records
	operations := []BulkOperation{
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeMX, "mail.example.com", 1800, 10)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().NoError(err)

	// Verify records were added
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 3)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_NonAtomic() {
	// Configure mock to be non-atomic
	s.mock.isAtomic = false

	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Mixed operations: add, update, delete
	operations := []BulkOperation{
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("mail", dnsrecord.RecordTypeA, "192.168.1.3", 1800, 0)),
		},
		{
			Action: BulkActionUpdate,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.100", 3600, 0)),
		},
		{
			Action: BulkActionDelete,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().NoError(err)

	// Verify final state
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 2)

	// Check that @ was updated
	var rootRecord *dnsrecord.Record
	for i := range records {
		if records[i].HostName == "@" {
			rootRecord = &records[i]
			break
		}
	}
	s.Require().NotNil(rootRecord)
	s.Require().Equal("192.168.1.100", rootRecord.Address)

	// Check that mail was added
	var mailRecord *dnsrecord.Record
	for i := range records {
		if records[i].HostName == "mail" {
			mailRecord = &records[i]
			break
		}
	}
	s.Require().NotNil(mailRecord)
	s.Require().Equal("192.168.1.3", mailRecord.Address)

	// Check that www was deleted
	var wwwRecord *dnsrecord.Record
	for i := range records {
		if records[i].HostName == "www" {
			wwwRecord = &records[i]
			break
		}
	}
	s.Require().Nil(wwwRecord)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_Update() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Bulk update records
	operations := []BulkOperation{
		{
			Action: BulkActionUpdate,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.100", 3600, 0)),
		},
		{
			Action: BulkActionUpdate,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.200", 3600, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().NoError(err)

	// Verify records were updated
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 2)
	s.Require().Equal("192.168.1.100", records[0].Address)
	s.Require().Equal("192.168.1.200", records[1].Address)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_Delete() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("mail", dnsrecord.RecordTypeA, "192.168.1.3", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Bulk delete records
	operations := []BulkOperation{
		{
			Action: BulkActionDelete,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		},
		{
			Action: BulkActionDelete,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().NoError(err)

	// Verify records were deleted
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal("mail", records[0].HostName)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_MixedOperations() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Mixed operations: add, update, delete
	operations := []BulkOperation{
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("mail", dnsrecord.RecordTypeA, "192.168.1.3", 1800, 0)),
		},
		{
			Action: BulkActionUpdate,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.100", 3600, 0)),
		},
		{
			Action: BulkActionDelete,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().NoError(err)

	// Verify final state
	records, err := s.service.GetRecords(s.ctx, domain)
	s.Require().NoError(err)
	s.Require().Len(records, 2)

	// Check that @ was updated
	var rootRecord *dnsrecord.Record
	for i := range records {
		if records[i].HostName == "@" {
			rootRecord = &records[i]
			break
		}
	}
	s.Require().NotNil(rootRecord)
	s.Require().Equal("192.168.1.100", rootRecord.Address)

	// Check that mail was added
	var mailRecord *dnsrecord.Record
	for i := range records {
		if records[i].HostName == "mail" {
			mailRecord = &records[i]
			break
		}
	}
	s.Require().NotNil(mailRecord)
	s.Require().Equal("192.168.1.3", mailRecord.Address)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_InvalidAction() {
	domain := testutil.ValidDomainFixture()

	operations := []BulkOperation{
		{
			Action: "invalid_action",
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_InvalidRecord() {
	domain := testutil.ValidDomainFixture()

	operations := []BulkOperation{
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_UpdateNotFound() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Try to update non-existent record
	operations := []BulkOperation{
		{
			Action: BulkActionUpdate,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_DeleteNotFound() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	// Try to delete non-existent record
	operations := []BulkOperation{
		{
			Action: BulkActionDelete,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().Error(err)
}

func (s *ServiceTestSuite) TestService_BulkUpdate_GetRecordsError() {
	domain := testutil.ValidDomainFixture()
	expectedError := errors.New("provider error")
	s.mock.getRecordsError = expectedError

	operations := []BulkOperation{
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().Error(err)
	// BulkUpdate wraps the error, so check that it contains the original error
	s.Require().Contains(err.Error(), "failed to get existing records")
	s.Require().Contains(err.Error(), "provider error")
}

func (s *ServiceTestSuite) TestService_BulkUpdate_SetRecordsError() {
	domain := testutil.ValidDomainFixture()

	// Setup existing records
	existingRecords := []dnsrecord.Record{
		convertDNSRecord(testutil.DNSRecordFixtureWithValues("@", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0)),
	}
	s.mock.records[domain] = existingRecords

	expectedError := errors.New("provider error")
	s.mock.setRecordsError = expectedError

	operations := []BulkOperation{
		{
			Action: BulkActionAdd,
			Record: convertDNSRecord(testutil.DNSRecordFixtureWithValues("www", dnsrecord.RecordTypeA, "192.168.1.2", 1800, 0)),
		},
	}

	err := s.service.BulkUpdate(s.ctx, domain, operations)
	s.Require().Error(err)
	s.Require().Equal(expectedError, err)
}

func (s *ServiceTestSuite) TestService_AddRecord_ValidationError() {
	domain := testutil.ValidDomainFixture()

	// Try to add invalid record (empty hostname)
	invalidRecord := convertDNSRecord(testutil.DNSRecordFixtureWithValues("", dnsrecord.RecordTypeA, "192.168.1.1", 1800, 0))
	err := s.service.AddRecord(s.ctx, domain, invalidRecord)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid record")
}

func (s *ServiceTestSuite) TestService_NewServiceWithProviderName() {
	// Register a mock provider
	mock := newMockProvider("test-provider")
	err := provider.Register(mock)
	s.Require().NoError(err)
	defer provider.Unregister("test-provider")

	// Create service with provider name
	service, err := NewServiceWithProviderName("test-provider")
	s.Require().NoError(err)
	s.Require().NotNil(service)
}

func (s *ServiceTestSuite) TestService_NewServiceWithProviderName_NotFound() {
	// Try to create service with non-existent provider
	service, err := NewServiceWithProviderName("non-existent-provider")
	s.Require().Error(err)
	s.Require().Nil(service)
}

// convertDNSRecord converts testutil DNS record to dnsrecord.Record
func convertDNSRecord(fixture testutil.TestDNSRecord) dnsrecord.Record {
	return dnsrecord.Record{
		HostName:   fixture.HostName,
		RecordType: fixture.RecordType,
		Address:    fixture.Address,
		TTL:        fixture.TTL,
		MXPref:     fixture.MXPref,
	}
}
