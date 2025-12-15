package dns

import (
	"fmt"
	"strings"

	"zonekit/pkg/client"
	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dns/provider/namecheap"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/errors"
)

// Service provides DNS record management operations
type Service struct {
	provider provider.Provider
}

// NewService creates a new DNS service with Namecheap provider
func NewService(client *client.Client) *Service {
	// Register Namecheap provider
	_ = namecheap.Register(client)

	// Get the provider from registry
	dnsProvider, _ := provider.Get("namecheap")

	return &Service{
		provider: dnsProvider,
	}
}

// NewServiceWithProvider creates a new DNS service with a specific provider
func NewServiceWithProvider(dnsProvider provider.Provider) *Service {
	return &Service{
		provider: dnsProvider,
	}
}

// NewServiceWithProviderName creates a new DNS service using a provider by name
func NewServiceWithProviderName(providerName string) (*Service, error) {
	dnsProvider, err := provider.Get(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS provider %s: %w", providerName, err)
	}

	return &Service{
		provider: dnsProvider,
	}, nil
}


// GetRecords retrieves all DNS records for a domain
func (s *Service) GetRecords(domainName string) ([]dnsrecord.Record, error) {
	return s.provider.GetRecords(domainName)
}

// SetRecords sets DNS records for a domain (replaces all existing records)
func (s *Service) SetRecords(domainName string, records []dnsrecord.Record) error {
	return s.provider.SetRecords(domainName, records)
}

// AddRecord adds a single DNS record to a domain
func (s *Service) AddRecord(domainName string, record dnsrecord.Record) error {
	// Validate record before adding
	if err := s.ValidateRecord(record); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}

	// Get existing records
	existingRecords, err := s.GetRecords(domainName)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	// Add new record
	allRecords := append(existingRecords, record)

	// Set all records
	return s.SetRecords(domainName, allRecords)
}

// UpdateRecord updates a DNS record by hostname and type
func (s *Service) UpdateRecord(domainName string, hostname, recordType string, newRecord dnsrecord.Record) error {
	// Get existing records
	existingRecords, err := s.GetRecords(domainName)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	// Find and update the record
	found := false
	for i, record := range existingRecords {
		if record.HostName == hostname && record.RecordType == recordType {
			existingRecords[i] = newRecord
			found = true
			break
		}
	}

	if !found {
		return errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", hostname, recordType))
	}

	// Set all records
	return s.SetRecords(domainName, existingRecords)
}

// DeleteRecord removes a DNS record by hostname and type
func (s *Service) DeleteRecord(domainName string, hostname, recordType string) error {
	// Get existing records
	existingRecords, err := s.GetRecords(domainName)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	// Filter out the record to delete
	var filteredRecords []dnsrecord.Record
	found := false
	for _, record := range existingRecords {
		if record.HostName == hostname && record.RecordType == recordType {
			found = true
			continue
		}
		filteredRecords = append(filteredRecords, record)
	}

	if !found {
		return errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", hostname, recordType))
	}

	// Set remaining records
	return s.SetRecords(domainName, filteredRecords)
}

// DeleteAllRecords removes all DNS records for a domain
func (s *Service) DeleteAllRecords(domainName string) error {
	return s.SetRecords(domainName, []dnsrecord.Record{})
}

// GetRecordsByType filters records by type
func (s *Service) GetRecordsByType(domainName string, recordType string) ([]dnsrecord.Record, error) {
	allRecords, err := s.GetRecords(domainName)
	if err != nil {
		return nil, err
	}

	var filteredRecords []dnsrecord.Record
	for _, record := range allRecords {
		if record.RecordType == recordType {
			filteredRecords = append(filteredRecords, record)
		}
	}

	return filteredRecords, nil
}

// ValidateRecord validates a DNS record before adding/updating
func (s *Service) ValidateRecord(record dnsrecord.Record) error {
	if record.HostName == "" {
		return errors.NewInvalidInput("hostname", "cannot be empty")
	}

	if record.RecordType == "" {
		return errors.NewInvalidInput("record_type", "cannot be empty")
	}

	if record.Address == "" {
		return errors.NewInvalidInput("address", "cannot be empty")
	}

	// Validate record type
	validTypes := []string{dnsrecord.RecordTypeA, dnsrecord.RecordTypeAAAA, dnsrecord.RecordTypeCNAME, dnsrecord.RecordTypeMX, dnsrecord.RecordTypeTXT, dnsrecord.RecordTypeNS, dnsrecord.RecordTypeSRV}
	isValid := false
	for _, validType := range validTypes {
		if record.RecordType == validType {
			isValid = true
			break
		}
	}

	if !isValid {
		return errors.NewInvalidInput("record_type", fmt.Sprintf("invalid type: %s (must be one of: %s)", record.RecordType, strings.Join(validTypes, ", ")))
	}

	// Validate TTL if provided
	if record.TTL > 0 {
		if record.TTL < MinTTL {
			return errors.NewInvalidInput("ttl", fmt.Sprintf("must be at least %d", MinTTL))
		}
		if record.TTL > MaxTTL {
			return errors.NewInvalidInput("ttl", fmt.Sprintf("must be at most %d", MaxTTL))
		}
	}

	// Validate MX preference if provided
	if record.MXPref > 0 {
		if record.MXPref < MinMXPref {
			return errors.NewInvalidInput("mx_pref", fmt.Sprintf("must be at least %d", MinMXPref))
		}
		if record.MXPref > MaxMXPref {
			return errors.NewInvalidInput("mx_pref", fmt.Sprintf("must be at most %d", MaxMXPref))
		}
	}

	// Type-specific validation
	switch record.RecordType {
	case dnsrecord.RecordTypeA:
		if err := ValidateIPv4(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("A record must have valid IPv4 address: %v", err))
		}
	case dnsrecord.RecordTypeAAAA:
		if err := ValidateIPv6(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("AAAA record must have valid IPv6 address: %v", err))
		}
	case dnsrecord.RecordTypeMX:
		if record.MXPref <= 0 {
			return errors.NewInvalidInput("mx_pref", "MX records must have a priority value")
		}
		// MX address should be a valid hostname
		if err := ValidateHostname(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("MX record must have valid hostname: %v", err))
		}
	case dnsrecord.RecordTypeCNAME:
		// CNAME address should be a valid hostname
		if err := ValidateHostname(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("CNAME record must have valid hostname: %v", err))
		}
	case dnsrecord.RecordTypeNS:
		// NS address should be a valid hostname
		if err := ValidateHostname(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("NS record must have valid hostname: %v", err))
		}
	}

	return nil
}

// BulkOperation represents a bulk DNS operation
type BulkOperation struct {
	Action string // Use BulkActionAdd, BulkActionUpdate, or BulkActionDelete constants
	Record dnsrecord.Record
}

// BulkUpdate performs multiple DNS operations in a single API call
func (s *Service) BulkUpdate(domainName string, operations []BulkOperation) error {
	// Get existing records
	existingRecords, err := s.GetRecords(domainName)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	records := make([]dnsrecord.Record, len(existingRecords))
	copy(records, existingRecords)

	// Apply operations
	for _, op := range operations {
		switch op.Action {
		case BulkActionAdd:
			if err := s.ValidateRecord(op.Record); err != nil {
				return fmt.Errorf("invalid record for add operation: %w", err)
			}
			records = append(records, op.Record)

		case BulkActionUpdate:
			if err := s.ValidateRecord(op.Record); err != nil {
				return fmt.Errorf("invalid record for update operation: %w", err)
			}
			found := false
			for i, record := range records {
				if record.HostName == op.Record.HostName && record.RecordType == op.Record.RecordType {
					records[i] = op.Record
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("record not found for update: %s %s", op.Record.HostName, op.Record.RecordType)
			}

		case BulkActionDelete:
			var filteredRecords []dnsrecord.Record
			found := false
			for _, record := range records {
				if record.HostName == op.Record.HostName && record.RecordType == op.Record.RecordType {
					found = true
					continue
				}
				filteredRecords = append(filteredRecords, record)
			}
			if !found {
				return errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", op.Record.HostName, op.Record.RecordType))
			}
			records = filteredRecords

		default:
			return errors.NewInvalidInput("action", fmt.Sprintf("invalid bulk operation action: %s (must be one of: %s, %s, %s)", op.Action, BulkActionAdd, BulkActionUpdate, BulkActionDelete))
		}
	}

	// Set all records
	return s.SetRecords(domainName, records)
}

func parseDomain(fullDomain string) (string, string) {
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 2 {
		return fullDomain, ""
	}

	// Handle subdomains - take the last two parts as domain and TLD
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
	}

	return parts[0], parts[1]
}
