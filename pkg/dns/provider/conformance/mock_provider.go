package conformance

import (
	"context"
	"fmt"
	"sync"
	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"

	"github.com/google/uuid"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	zones   map[string]provider.Zone
	records map[string]map[string]dnsrecord.Record // zoneID -> recordID -> Record
	mu      sync.RWMutex
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		zones:   make(map[string]provider.Zone),
		records: make(map[string]map[string]dnsrecord.Record),
	}
}

// Helper to seed data
func (m *MockProvider) AddZone(zone provider.Zone) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zones[zone.Name] = zone
    if _, ok := m.records[zone.ID]; !ok {
        m.records[zone.ID] = make(map[string]dnsrecord.Record)
    }
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportsRecordID:      true,
		SupportsBulkReplace:   true,
		SupportsZoneDiscovery: true,
		SupportedRecordTypes:  []string{"A", "TXT", "CNAME", "MX", "SRV"},
	}
}

func (m *MockProvider) ListZones(ctx context.Context) ([]provider.Zone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var zones []provider.Zone
	for _, z := range m.zones {
		zones = append(zones, z)
	}
	return zones, nil
}

func (m *MockProvider) GetZone(ctx context.Context, domain string) (*provider.Zone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	z, ok := m.zones[domain]
	if !ok {
		return nil, fmt.Errorf("zone not found")
	}
	return &z, nil
}

func (m *MockProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

    records := []dnsrecord.Record{}
    if zoneRecords, ok := m.records[zoneID]; ok {
        for _, r := range zoneRecords {
            records = append(records, r)
        }
    }
	return records, nil
}

func (m *MockProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

    if _, ok := m.records[zoneID]; !ok {
        // Ensure zone record map exists
        m.records[zoneID] = make(map[string]dnsrecord.Record)
    }

	record.ID = uuid.New().String()
	m.records[zoneID][record.ID] = record
	return &record, nil
}

func (m *MockProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.records[zoneID]; !ok {
		return nil, fmt.Errorf("zone not found")
	}

    if _, ok := m.records[zoneID][recordID]; !ok {
        return nil, fmt.Errorf("record not found")
    }

    record.ID = recordID // Ensure ID is preserved
	m.records[zoneID][recordID] = record
	return &record, nil
}

func (m *MockProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.records[zoneID]; !ok {
		return fmt.Errorf("zone not found")
	}

    if _, ok := m.records[zoneID][recordID]; !ok {
        return fmt.Errorf("record not found")
    }

	delete(m.records[zoneID], recordID)
	return nil
}

func (m *MockProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

    // Clear existing
	m.records[zoneID] = make(map[string]dnsrecord.Record)

    for _, r := range records {
        if r.ID == "" {
            r.ID = uuid.New().String()
        }
        m.records[zoneID][r.ID] = r
    }
	return nil
}
