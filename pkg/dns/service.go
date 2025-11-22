package dns

import (
	"fmt"
	"strings"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"namecheap-dns-manager/pkg/client"
	"namecheap-dns-manager/pkg/errors"
	"namecheap-dns-manager/pkg/pointer"
)

// Service provides DNS record management operations
type Service struct {
	client *client.Client
}

// NewService creates a new DNS service
func NewService(client *client.Client) *Service {
	return &Service{
		client: client,
	}
}

// Record represents a DNS record
type Record struct {
	HostName   string
	RecordType string
	Address    string
	TTL        int
	MXPref     int
}

// RecordType constants
const (
	RecordTypeA     = "A"
	RecordTypeAAAA  = "AAAA"
	RecordTypeCNAME = "CNAME"
	RecordTypeMX    = "MX"
	RecordTypeTXT   = "TXT"
	RecordTypeNS    = "NS"
	RecordTypeSRV   = "SRV"
)

// GetRecords retrieves all DNS records for a domain
func (s *Service) GetRecords(domainName string) ([]Record, error) {
	nc := s.client.GetNamecheapClient()

	resp, err := nc.DomainsDNS.GetHosts(domainName)
	if err != nil {
		return nil, errors.NewAPI("GetHosts", fmt.Sprintf("failed to get DNS records for %s", domainName), err)
	}

	// Safety check for nil response
	if resp == nil || resp.DomainDNSGetHostsResult == nil || resp.DomainDNSGetHostsResult.Hosts == nil {
		return []Record{}, nil
	}

	records := make([]Record, 0, len(*resp.DomainDNSGetHostsResult.Hosts))
	for _, host := range *resp.DomainDNSGetHostsResult.Hosts {
		record := Record{
			HostName:   pointer.String(host.Name),
			RecordType: pointer.String(host.Type),
			Address:    pointer.String(host.Address),
			TTL:        pointer.Int(host.TTL),
			MXPref:     pointer.Int(host.MXPref),
		}
		records = append(records, record)
	}

	return records, nil
}

// SetRecords sets DNS records for a domain (replaces all existing records)
func (s *Service) SetRecords(domainName string, records []Record) error {
	nc := s.client.GetNamecheapClient()

	// Convert records to Namecheap format
	hostRecords := make([]namecheap.DomainsDNSHostRecord, len(records))
	hasMXRecords := false
	for i, record := range records {
		hostRecord := namecheap.DomainsDNSHostRecord{
			HostName:   namecheap.String(record.HostName),
			RecordType: namecheap.String(record.RecordType),
			Address:    namecheap.String(record.Address),
		}

		if record.TTL > 0 {
			hostRecord.TTL = namecheap.Int(record.TTL)
		}

		if record.MXPref > 0 {
			hostRecord.MXPref = namecheap.UInt8(uint8(record.MXPref))
		}

		// Check if this is an MX record
		if record.RecordType == RecordTypeMX {
			hasMXRecords = true
		}

		hostRecords[i] = hostRecord
	}

	// Build SetHostsArgs
	args := &namecheap.DomainsDNSSetHostsArgs{
		Domain:  namecheap.String(domainName),
		Records: &hostRecords,
	}

	// Set EmailType to MX if there are any MX records
	// This is required by the Namecheap API when MX records are present
	if hasMXRecords {
		args.EmailType = namecheap.String(EmailTypeMX)
	}

	_, err := nc.DomainsDNS.SetHosts(args)
	if err != nil {
		return errors.NewAPI("SetHosts", fmt.Sprintf("failed to set DNS records for %s", domainName), err)
	}

	return nil
}

// AddRecord adds a single DNS record to a domain
func (s *Service) AddRecord(domainName string, record Record) error {
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
func (s *Service) UpdateRecord(domainName string, hostname, recordType string, newRecord Record) error {
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
	var filteredRecords []Record
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
	return s.SetRecords(domainName, []Record{})
}

// GetRecordsByType filters records by type
func (s *Service) GetRecordsByType(domainName string, recordType string) ([]Record, error) {
	allRecords, err := s.GetRecords(domainName)
	if err != nil {
		return nil, err
	}

	var filteredRecords []Record
	for _, record := range allRecords {
		if record.RecordType == recordType {
			filteredRecords = append(filteredRecords, record)
		}
	}

	return filteredRecords, nil
}

// ValidateRecord validates a DNS record before adding/updating
func (s *Service) ValidateRecord(record Record) error {
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
	validTypes := []string{RecordTypeA, RecordTypeAAAA, RecordTypeCNAME, RecordTypeMX, RecordTypeTXT, RecordTypeNS, RecordTypeSRV}
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
	case RecordTypeA:
		if err := ValidateIPv4(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("A record must have valid IPv4 address: %v", err))
		}
	case RecordTypeAAAA:
		if err := ValidateIPv6(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("AAAA record must have valid IPv6 address: %v", err))
		}
	case RecordTypeMX:
		if record.MXPref <= 0 {
			return errors.NewInvalidInput("mx_pref", "MX records must have a priority value")
		}
		// MX address should be a valid hostname
		if err := ValidateHostname(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("MX record must have valid hostname: %v", err))
		}
	case RecordTypeCNAME:
		// CNAME address should be a valid hostname
		if err := ValidateHostname(record.Address); err != nil {
			return errors.NewInvalidInput("address", fmt.Sprintf("CNAME record must have valid hostname: %v", err))
		}
	case RecordTypeNS:
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
	Record Record
}

// BulkUpdate performs multiple DNS operations in a single API call
func (s *Service) BulkUpdate(domainName string, operations []BulkOperation) error {
	// Get existing records
	existingRecords, err := s.GetRecords(domainName)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	records := make([]Record, len(existingRecords))
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
			var filteredRecords []Record
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
