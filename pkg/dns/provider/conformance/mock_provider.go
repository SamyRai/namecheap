package conformance

import (
	"context"
	"fmt"

	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	Zones   map[string]provider.Zone
	Records map[string]map[string]dnsrecord.Record // zoneID -> recordID -> Record
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		Zones:   make(map[string]provider.Zone),
		Records: make(map[string]map[string]dnsrecord.Record),
	}
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) ListZones(ctx context.Context) ([]provider.Zone, error) {
	var zones []provider.Zone
	for _, z := range m.Zones {
		zones = append(zones, z)
	}
	return zones, nil
}

func (m *MockProvider) GetZone(ctx context.Context, zoneID string) (provider.Zone, error) {
	z, ok := m.Zones[zoneID]
	if !ok {
		return provider.Zone{}, fmt.Errorf("zone not found")
	}
	return z, nil
}

func (m *MockProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	if _, ok := m.Zones[zoneID]; !ok {
		return nil, fmt.Errorf("zone not found")
	}
	var records []dnsrecord.Record
	if zoneRecords, ok := m.Records[zoneID]; ok {
		for _, r := range zoneRecords {
			records = append(records, r)
		}
	}
	return records, nil
}

func (m *MockProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	if _, ok := m.Zones[zoneID]; !ok {
		return dnsrecord.Record{}, fmt.Errorf("zone not found")
	}
	if m.Records[zoneID] == nil {
		m.Records[zoneID] = make(map[string]dnsrecord.Record)
	}
	if record.ID == "" {
		record.ID = fmt.Sprintf("rec-%d", len(m.Records[zoneID])+1)
	}
	m.Records[zoneID][record.ID] = record
	return record, nil
}

func (m *MockProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	if _, ok := m.Zones[zoneID]; !ok {
		return dnsrecord.Record{}, fmt.Errorf("zone not found")
	}
	if _, ok := m.Records[zoneID][recordID]; !ok {
		return dnsrecord.Record{}, fmt.Errorf("record not found")
	}
	record.ID = recordID
	m.Records[zoneID][recordID] = record
	return record, nil
}

func (m *MockProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	if _, ok := m.Zones[zoneID]; !ok {
		return fmt.Errorf("zone not found")
	}
	if _, ok := m.Records[zoneID][recordID]; !ok {
		return fmt.Errorf("record not found")
	}
	delete(m.Records[zoneID], recordID)
	return nil
}

func (m *MockProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	if _, ok := m.Zones[zoneID]; !ok {
		return fmt.Errorf("zone not found")
	}
	m.Records[zoneID] = make(map[string]dnsrecord.Record)
	for _, r := range records {
		if r.ID == "" {
			r.ID = fmt.Sprintf("rec-%d", len(m.Records[zoneID])+1)
		}
		m.Records[zoneID][r.ID] = r
	}
	return nil
}

func (m *MockProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		CanListZones:    true,
		CanGetZone:      true,
		CanCreateRecord: true,
		CanUpdateRecord: true,
		CanDeleteRecord: true,
		CanBulkReplace:  true,
	}
}

func (m *MockProvider) Validate() error {
	return nil
}

var _ provider.Provider = (*MockProvider)(nil)
