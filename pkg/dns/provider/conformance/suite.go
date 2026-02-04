package conformance

import (
	"context"

	"github.com/stretchr/testify/suite"
	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

// ProviderFactory is a function that returns a configured provider and a cleanup function
type ProviderFactory func() (provider.Provider, func(), error)

// Suite is a conformance test suite for DNS providers
type Suite struct {
	suite.Suite
	ProviderFactory ProviderFactory
	Provider        provider.Provider
	Cleanup         func()
	Domain          string
	Ctx             context.Context
}

// SetupTest runs before each test
func (s *Suite) SetupTest() {
	var err error
	s.Provider, s.Cleanup, err = s.ProviderFactory()
	s.Require().NoError(err, "failed to create provider")
	s.Ctx = context.Background()
	s.Domain = "example.com"
}

// TearDownTest runs after each test
func (s *Suite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

// TestGetRecordsEmpty verifies that GetRecords returns empty list for new domain
func (s *Suite) TestGetRecordsEmpty() {
	records, err := s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Empty(records)
}

// TestAddRecord verifies that AddRecord works
func (s *Suite) TestAddRecord() {
	record := dnsrecord.Record{
		HostName:   "www",
		RecordType: "A",
		Address:    "192.168.1.1",
		TTL:        300,
	}

	err := s.Provider.AddRecord(s.Ctx, s.Domain, record)
	s.Require().NoError(err)

	records, err := s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal(record.HostName, records[0].HostName)
	s.Require().Equal(record.RecordType, records[0].RecordType)
	s.Require().Equal(record.Address, records[0].Address)
}

// TestUpdateRecord verifies that UpdateRecord works
func (s *Suite) TestUpdateRecord() {
	record := dnsrecord.Record{
		HostName:   "www",
		RecordType: "A",
		Address:    "192.168.1.1",
		TTL:        300,
	}

	// Add initial record
	err := s.Provider.AddRecord(s.Ctx, s.Domain, record)
	s.Require().NoError(err)

	// Update record
	record.Address = "192.168.1.2"
	record.TTL = 600

	// Note: We need to make sure the record ID is set if the provider requires it.
	// For conformance testing, we should probably fetch the record first to get its ID.
	records, err := s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	record.ID = records[0].ID

	err = s.Provider.UpdateRecord(s.Ctx, s.Domain, record)
	s.Require().NoError(err)

	// Verify update
	records, err = s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal("192.168.1.2", records[0].Address)
	s.Require().Equal(600, records[0].TTL)
}

// TestDeleteRecord verifies that DeleteRecord works
func (s *Suite) TestDeleteRecord() {
	record := dnsrecord.Record{
		HostName:   "www",
		RecordType: "A",
		Address:    "192.168.1.1",
		TTL:        300,
	}

	// Add initial record
	err := s.Provider.AddRecord(s.Ctx, s.Domain, record)
	s.Require().NoError(err)

	// Fetch to get ID if needed
	records, err := s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	record = records[0]

	// Delete record
	err = s.Provider.DeleteRecord(s.Ctx, s.Domain, record)
	s.Require().NoError(err)

	// Verify deletion
	records, err = s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Empty(records)
}

// TestSetRecords verifies that SetRecords replaces all records
func (s *Suite) TestSetRecords() {
	initialRecords := []dnsrecord.Record{
		{
			HostName:   "www",
			RecordType: "A",
			Address:    "192.168.1.1",
			TTL:        300,
		},
		{
			HostName:   "api",
			RecordType: "CNAME",
			Address:    "www.example.com",
			TTL:        300,
		},
	}

	// Set initial records
	err := s.Provider.SetRecords(s.Ctx, s.Domain, initialRecords)
	s.Require().NoError(err)

	// Verify initial state
	records, err := s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Len(records, 2)

	// Replace with new records
	newRecords := []dnsrecord.Record{
		{
			HostName:   "mail",
			RecordType: "MX",
			Address:    "mail.example.com",
			MXPref:     10,
			TTL:        300,
		},
	}

	err = s.Provider.SetRecords(s.Ctx, s.Domain, newRecords)
	s.Require().NoError(err)

	// Verify replacement
	records, err = s.Provider.GetRecords(s.Ctx, s.Domain)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal("mail", records[0].HostName)
	s.Require().Equal("MX", records[0].RecordType)
}

// TestUpdateRecordNotFound verifies error when updating non-existent record
func (s *Suite) TestUpdateRecordNotFound() {
	record := dnsrecord.Record{
		HostName:   "nonexistent",
		RecordType: "A",
		Address:    "1.1.1.1",
		ID:         "missing-id",
	}

	err := s.Provider.UpdateRecord(s.Ctx, s.Domain, record)
	s.Require().Error(err)
	// We don't enforce specific error types yet, but it should fail
}

// TestDeleteRecordNotFound verifies error when deleting non-existent record
func (s *Suite) TestDeleteRecordNotFound() {
	record := dnsrecord.Record{
		HostName:   "nonexistent",
		RecordType: "A",
		Address:    "1.1.1.1",
		ID:         "missing-id",
	}

	// Note: DeleteRecord might return nil if record doesn't exist (idempotent),
	// or error. The interface doesn't strictly define this yet.
	// However, if we pass a specific ID that is missing, REST providers typically 404.
	// For now, we'll log the result but not strictly require error unless we define that behavior.
	err := s.Provider.DeleteRecord(s.Ctx, s.Domain, record)
	if err != nil {
		s.T().Logf("DeleteRecord returned error for missing record: %v", err)
	}
}

// Helper to create a pointer to string
func String(s string) *string {
	return &s
}

// Helper to create a pointer to int
func Int(i int) *int {
	return &i
}
