package conformance

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

// MockProvider is an in-memory provider for conformance testing
type MockProvider struct {
	mu           sync.RWMutex
	records      map[string][]dnsrecord.Record // domain -> records
	capabilities provider.ProviderCapabilities
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		records: make(map[string][]dnsrecord.Record),
		capabilities: provider.ProviderCapabilities{
			IsBulkReplaceAtomic: true, // Default to true, can be changed
		},
	}
}

// Name returns the provider name
func (p *MockProvider) Name() string {
	return "mock"
}

// Capabilities returns the provider capabilities
func (p *MockProvider) Capabilities() provider.ProviderCapabilities {
	return p.capabilities
}

// SetCapabilities sets the capabilities for testing
func (p *MockProvider) SetCapabilities(caps provider.ProviderCapabilities) {
	p.capabilities = caps
}

// GetRecords retrieves all DNS records for a domain
func (p *MockProvider) GetRecords(ctx context.Context, domainName string) ([]dnsrecord.Record, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	records, ok := p.records[domainName]
	if !ok {
		return []dnsrecord.Record{}, nil
	}

	// Return a copy to avoid race conditions if caller modifies the slice
	result := make([]dnsrecord.Record, len(records))
	copy(result, records)
	return result, nil
}

// SetRecords sets DNS records for a domain
func (p *MockProvider) SetRecords(ctx context.Context, domainName string, records []dnsrecord.Record) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Assign IDs to new records if missing
	newRecords := make([]dnsrecord.Record, len(records))
	for i, r := range records {
		if r.ID == "" {
			r.ID = uuid.New().String()
		}
		newRecords[i] = r
	}

	p.records[domainName] = newRecords
	return nil
}

// AddRecord adds a single record
func (p *MockProvider) AddRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	p.records[domainName] = append(p.records[domainName], record)
	return nil
}

// UpdateRecord updates a single record
func (p *MockProvider) UpdateRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	records, ok := p.records[domainName]
	if !ok {
		return fmt.Errorf("record not found: domain %s does not exist", domainName)
	}

	found := false
	for i, r := range records {
		// Match by ID if present, otherwise by HostName+Type (simplified)
		if (record.ID != "" && r.ID == record.ID) || (record.ID == "" && r.HostName == record.HostName && r.RecordType == record.RecordType) {
			// Update the record
			// Preserve ID if not provided in update
			if record.ID == "" {
				record.ID = r.ID
			}
			records[i] = record
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("record not found")
	}

	p.records[domainName] = records
	return nil
}

// DeleteRecord deletes a single record
func (p *MockProvider) DeleteRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	records, ok := p.records[domainName]
	if !ok {
		return fmt.Errorf("record not found: domain %s does not exist", domainName)
	}

	newRecords := make([]dnsrecord.Record, 0, len(records))
	found := false
	for _, r := range records {
		// Match by ID if present, otherwise by HostName+Type
		if (record.ID != "" && r.ID == record.ID) || (record.ID == "" && r.HostName == record.HostName && r.RecordType == record.RecordType) {
			found = true
			continue
		}
		newRecords = append(newRecords, r)
	}

	if !found {
		return fmt.Errorf("record not found")
	}

	p.records[domainName] = newRecords
	return nil
}

// Validate checks configuration
func (p *MockProvider) Validate() error {
	return nil
}
