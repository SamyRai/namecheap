package dns

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.glpx.pro/zonekit/pkg/client"
	"go.glpx.pro/zonekit/pkg/dns/provider"
	"go.glpx.pro/zonekit/pkg/dns/provider/namecheap"
	"go.glpx.pro/zonekit/pkg/dnsrecord"
	"go.glpx.pro/zonekit/pkg/errors"
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

// resolveZoneID resolves a domain name to a zone ID
func (s *Service) resolveZoneID(ctx context.Context, domainName string) (string, error) {
	// 1. Try GetZone assuming ID == domainName
	if s.provider.Capabilities().CanGetZone {
		z, err := s.provider.GetZone(ctx, domainName)
		if err == nil {
			return z.ID, nil
		}
	}

	// 2. ListZones
	if s.provider.Capabilities().CanListZones {
		zones, err := s.provider.ListZones(ctx)
		if err != nil {
			// Don't fail here, try fallback
		} else {
			for _, z := range zones {
				// Basic matching
				if strings.EqualFold(z.Name, domainName) || strings.EqualFold(z.Name, domainName+".") {
					return z.ID, nil
				}
			}
		}
	}

	// Fallback: use domainName as ID
	return domainName, nil
}

// GetRecords retrieves all DNS records for a domain
func (s *Service) GetRecords(domainName string) ([]dnsrecord.Record, error) {
	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return nil, err
	}
	return s.provider.ListRecords(ctx, zoneID)
}

// SetRecords sets DNS records for a domain (replaces all existing records)
func (s *Service) SetRecords(domainName string, records []dnsrecord.Record) error {
	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return err
	}
	return s.provider.BulkReplaceRecords(ctx, zoneID, records)
}

// AddRecord adds a single DNS record to a domain
func (s *Service) AddRecord(domainName string, record dnsrecord.Record) error {
	// Validate record before adding
	if err := s.ValidateRecord(record); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}

	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return err
	}

	if s.provider.Capabilities().CanCreateRecord {
		_, err := s.provider.CreateRecord(ctx, zoneID, record)
		return err
	}

	// Fallback: Get existing records
	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	// Add new record
	allRecords := append(existingRecords, record)

	// Set all records
	return s.provider.BulkReplaceRecords(ctx, zoneID, allRecords)
}

// UpdateRecord updates a DNS record by hostname and type
func (s *Service) UpdateRecord(domainName string, hostname, recordType string, newRecord dnsrecord.Record) error {
	return s.UpdateRecordMatching(domainName, hostname, recordType, "", newRecord)
}

// UpdateRecordMatching updates the DNS record identified by hostname and type,
// narrowed to the record whose current value equals matchValue when that is
// non-empty.
//
// (hostname, type) is NOT a unique key in DNS. An apex routinely carries
// several TXT records at once — SPF, DMARC, and provider verification tokens
// all live at `@`. Selecting one of them arbitrarily rewrites a record the
// caller never named and destroys its previous contents, which is
// unrecoverable without an external backup. So an ambiguous match is reported
// as an error listing the candidates, rather than resolved by guessing; pass
// matchValue to choose one deliberately.
func (s *Service) UpdateRecordMatching(domainName string, hostname, recordType, matchValue string, newRecord dnsrecord.Record) error {
	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return err
	}

	// Find the record to get ID
	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	foundIndex, err := findRecordMatch(existingRecords, hostname, recordType, matchValue)
	if err != nil {
		return err
	}

	recordID := existingRecords[foundIndex].ID

	if s.provider.Capabilities().CanUpdateRecord && recordID != "" {
		_, err := s.provider.UpdateRecord(ctx, zoneID, recordID, newRecord)
		return err
	}

	// Fallback to bulk replace
	existingRecords[foundIndex] = newRecord
	return s.provider.BulkReplaceRecords(ctx, zoneID, existingRecords)
}

// findRecordMatch narrows existingRecords to the single record identified by
// hostname+recordType, optionally disambiguated by matchValue (its current
// Address). It never mutates existingRecords or calls the provider.
//
// (hostname, type) is not a unique key in DNS: an apex can carry several TXT
// records at once (SPF, DMARC, verification tokens...). Rather than guess
// which one a caller meant, an ambiguous match is reported as an error
// listing the candidates so the caller can pass matchValue to choose one
// deliberately. See the UpdateRecordMatching doc comment for the incident
// this refusal was added for.
func findRecordMatch(existingRecords []dnsrecord.Record, hostname, recordType, matchValue string) (int, error) {
	matches := make([]int, 0, 1)
	for i, record := range existingRecords {
		if record.HostName != hostname || record.RecordType != recordType {
			continue
		}
		if matchValue != "" && record.Address != matchValue {
			continue
		}
		matches = append(matches, i)
	}

	switch len(matches) {
	case 0:
		if matchValue != "" {
			return -1, errors.NewNotFound("DNS record",
				fmt.Sprintf("%s %s with value %q", hostname, recordType, matchValue))
		}
		return -1, errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", hostname, recordType))
	case 1:
		return matches[0], nil
	default:
		values := make([]string, 0, len(matches))
		for _, i := range matches {
			values = append(values, strconv.Quote(existingRecords[i].Address))
		}
		return -1, errors.NewInvalidInput("hostname", fmt.Sprintf(
			"%d records match %s %s; refusing to guess which to replace. "+
				"Re-run with --match-value to select one, or delete and re-add the set. Candidates: %s",
			len(matches), hostname, recordType, strings.Join(values, ", ")))
	}
}

// PlannedChange describes a single record mutation a mutating command would
// perform under --dry-run (O3). Before is nil for a create; After is nil for
// a delete.
type PlannedChange struct {
	Action string            `json:"action" yaml:"action"` // create|update|delete
	Before *dnsrecord.Record `json:"before,omitempty" yaml:"before,omitempty"`
	After  *dnsrecord.Record `json:"after,omitempty" yaml:"after,omitempty"`
}

// Planned change actions.
const (
	PlanActionCreate = "create"
	PlanActionUpdate = "update"
	PlanActionDelete = "delete"
)

// PlanAddRecord validates record and reports the change AddRecord would make,
// without calling the provider at all (an add never depends on the record's
// current state).
func (s *Service) PlanAddRecord(record dnsrecord.Record) (PlannedChange, error) {
	if err := s.ValidateRecord(record); err != nil {
		return PlannedChange{}, fmt.Errorf("invalid record: %w", err)
	}
	r := record
	return PlannedChange{Action: PlanActionCreate, After: &r}, nil
}

// PlanUpdateRecordMatching reports the change UpdateRecordMatching would
// make, without writing it. It still calls the provider's read-only
// ListRecords so it can apply the same ambiguity refusal the real update
// does, and so the "before" value in the plan is accurate.
func (s *Service) PlanUpdateRecordMatching(domainName, hostname, recordType, matchValue string, newRecord dnsrecord.Record) (PlannedChange, error) {
	if err := s.ValidateRecord(newRecord); err != nil {
		return PlannedChange{}, fmt.Errorf("invalid record: %w", err)
	}

	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return PlannedChange{}, err
	}

	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return PlannedChange{}, fmt.Errorf("failed to get existing records: %w", err)
	}

	idx, err := findRecordMatch(existingRecords, hostname, recordType, matchValue)
	if err != nil {
		return PlannedChange{}, err
	}

	before := existingRecords[idx]
	after := newRecord
	return PlannedChange{Action: PlanActionUpdate, Before: &before, After: &after}, nil
}

// PlanDeleteRecord reports the change DeleteRecord would make, without
// writing it.
func (s *Service) PlanDeleteRecord(domainName string, hostname, recordType string) (PlannedChange, error) {
	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return PlannedChange{}, err
	}

	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return PlannedChange{}, fmt.Errorf("failed to get existing records: %w", err)
	}

	for _, record := range existingRecords {
		if record.HostName == hostname && record.RecordType == recordType {
			r := record
			return PlannedChange{Action: PlanActionDelete, Before: &r}, nil
		}
	}

	return PlannedChange{}, errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", hostname, recordType))
}

// PlanClear reports the changes DeleteAllRecords would make, without writing
// them: one delete per record currently on the domain.
func (s *Service) PlanClear(domainName string) ([]PlannedChange, error) {
	records, err := s.GetRecords(domainName)
	if err != nil {
		return nil, err
	}

	plans := make([]PlannedChange, len(records))
	for i, r := range records {
		record := r
		plans[i] = PlannedChange{Action: PlanActionDelete, Before: &record}
	}
	return plans, nil
}

// DeleteRecord removes a DNS record by hostname and type
func (s *Service) DeleteRecord(domainName string, hostname, recordType string) error {
	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return err
	}

	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	var recordID string
	found := false
	var filteredRecords []dnsrecord.Record

	for _, record := range existingRecords {
		if record.HostName == hostname && record.RecordType == recordType {
			recordID = record.ID
			found = true
			continue
		}
		filteredRecords = append(filteredRecords, record)
	}

	if !found {
		return errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", hostname, recordType))
	}

	if s.provider.Capabilities().CanDeleteRecord && recordID != "" {
		return s.provider.DeleteRecord(ctx, zoneID, recordID)
	}

	return s.provider.BulkReplaceRecords(ctx, zoneID, filteredRecords)
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
	return validateRecordStatic(record)
}

// validateRecordStatic is the record-validation logic. It is a free function
// (not a Service method) so package-level helpers like applyBulkOperations
// can validate without needing a Service/provider; ValidateRecord delegates
// to it to keep the existing method signature.
func validateRecordStatic(record dnsrecord.Record) error {
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

	records, _, err := applyBulkOperations(existingRecords, operations)
	if err != nil {
		return err
	}

	// Set all records
	return s.SetRecords(domainName, records)
}

// PlanBulkUpdate reports the changes BulkUpdate would make, without writing
// them.
func (s *Service) PlanBulkUpdate(domainName string, operations []BulkOperation) ([]PlannedChange, error) {
	existingRecords, err := s.GetRecords(domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing records: %w", err)
	}

	_, plans, err := applyBulkOperations(existingRecords, operations)
	return plans, err
}

// applyBulkOperations simulates operations against existingRecords, returning
// the resulting record set and the list of changes made along the way. It
// never touches the provider, so BulkUpdate (which writes the result) and
// PlanBulkUpdate (which only reports it) share one implementation of "what
// do these operations mean".
func applyBulkOperations(existingRecords []dnsrecord.Record, operations []BulkOperation) ([]dnsrecord.Record, []PlannedChange, error) {
	records := make([]dnsrecord.Record, len(existingRecords))
	copy(records, existingRecords)

	var plans []PlannedChange
	var err error

	for _, op := range operations {
		var plan PlannedChange
		switch op.Action {
		case BulkActionAdd:
			records, plan, err = applyBulkAdd(records, op.Record)
		case BulkActionUpdate:
			records, plan, err = applyBulkUpdate(records, op.Record)
		case BulkActionDelete:
			records, plan, err = applyBulkDelete(records, op.Record)
		default:
			err = errors.NewInvalidInput("action", fmt.Sprintf("invalid bulk operation action: %s (must be one of: %s, %s, %s)", op.Action, BulkActionAdd, BulkActionUpdate, BulkActionDelete))
		}
		if err != nil {
			return nil, nil, err
		}
		plans = append(plans, plan)
	}

	return records, plans, nil
}

// applyBulkAdd is the BulkActionAdd step of applyBulkOperations: append
// record and report it as a planned create.
func applyBulkAdd(records []dnsrecord.Record, record dnsrecord.Record) ([]dnsrecord.Record, PlannedChange, error) {
	if err := validateRecordStatic(record); err != nil {
		return nil, PlannedChange{}, fmt.Errorf("invalid record for add operation: %w", err)
	}
	after := record
	records = append(records, record)
	return records, PlannedChange{Action: PlanActionCreate, After: &after}, nil
}

// applyBulkUpdate is the BulkActionUpdate step of applyBulkOperations:
// replace the first record matching record's hostname+type in place.
func applyBulkUpdate(records []dnsrecord.Record, record dnsrecord.Record) ([]dnsrecord.Record, PlannedChange, error) {
	if err := validateRecordStatic(record); err != nil {
		return nil, PlannedChange{}, fmt.Errorf("invalid record for update operation: %w", err)
	}
	for i := range records {
		if records[i].HostName == record.HostName && records[i].RecordType == record.RecordType {
			before := records[i]
			after := record
			records[i] = record
			return records, PlannedChange{Action: PlanActionUpdate, Before: &before, After: &after}, nil
		}
	}
	return nil, PlannedChange{}, fmt.Errorf("record not found for update: %s %s", record.HostName, record.RecordType)
}

// applyBulkDelete is the BulkActionDelete step of applyBulkOperations: remove
// the first record matching record's hostname+type, reporting the removed
// record's actual stored value (not just the hostname/type the caller
// searched by).
func applyBulkDelete(records []dnsrecord.Record, record dnsrecord.Record) ([]dnsrecord.Record, PlannedChange, error) {
	var filtered []dnsrecord.Record
	var removed *dnsrecord.Record
	for _, existing := range records {
		if removed == nil && existing.HostName == record.HostName && existing.RecordType == record.RecordType {
			before := existing
			removed = &before
			continue
		}
		filtered = append(filtered, existing)
	}
	if removed == nil {
		return nil, PlannedChange{}, errors.NewNotFound("DNS record", fmt.Sprintf("%s %s", record.HostName, record.RecordType))
	}
	return filtered, PlannedChange{Action: PlanActionDelete, Before: removed}, nil
}

// EnsureResult reports what Service.Ensure (or PlanEnsure) did, or would do.
type EnsureResult struct {
	Status string           `json:"status" yaml:"status"` // created|updated|unchanged
	Record dnsrecord.Record `json:"record" yaml:"record"`
}

// Ensure outcome statuses.
const (
	EnsureStatusCreated   = "created"
	EnsureStatusUpdated   = "updated"
	EnsureStatusUnchanged = "unchanged"
)

// multiValueRecordTypes are record types where a single hostname
// conventionally carries several independent, coexisting records (multiple
// MX with different priorities; multiple TXT for SPF/DMARC/verification
// tokens). For these, Ensure's identity is (hostname, type, value): a record
// is only touched if an existing one already has that exact value, otherwise
// a new one is added alongside its siblings rather than overwriting one of
// them. Every other type's identity is (hostname, type), matching normal DNS
// practice of at most one A/AAAA/CNAME/NS record per hostname.
var multiValueRecordTypes = map[string]bool{
	dnsrecord.RecordTypeMX:  true,
	dnsrecord.RecordTypeTXT: true,
}

// ensurePlan decides what Ensure must do to make record present with its
// current value/ttl/mx_pref, given the domain's existing records. It never
// touches the provider.
func ensurePlan(existingRecords []dnsrecord.Record, record dnsrecord.Record) (action string, matchIndex int, err error) {
	sameHostnameType := func(r dnsrecord.Record) bool {
		return r.HostName == record.HostName && r.RecordType == record.RecordType
	}
	unchanged := func(r dnsrecord.Record) bool {
		return r.Address == record.Address && r.TTL == record.TTL && r.MXPref == record.MXPref
	}

	if multiValueRecordTypes[record.RecordType] {
		for i, r := range existingRecords {
			if sameHostnameType(r) && r.Address == record.Address {
				if unchanged(r) {
					return EnsureStatusUnchanged, i, nil
				}
				return EnsureStatusUpdated, i, nil
			}
		}
		return EnsureStatusCreated, -1, nil
	}

	matches := make([]int, 0, 1)
	for i, r := range existingRecords {
		if sameHostnameType(r) {
			matches = append(matches, i)
		}
	}

	switch len(matches) {
	case 0:
		return EnsureStatusCreated, -1, nil
	case 1:
		i := matches[0]
		if unchanged(existingRecords[i]) {
			return EnsureStatusUnchanged, i, nil
		}
		return EnsureStatusUpdated, i, nil
	default:
		values := make([]string, 0, len(matches))
		for _, i := range matches {
			values = append(values, strconv.Quote(existingRecords[i].Address))
		}
		return "", -1, errors.NewInvalidInput("hostname", fmt.Sprintf(
			"%d records already match %s %s; ensure cannot decide which one to update. "+
				"Resolve the ambiguity first (dns update --match-value, or dns delete), then retry. Candidates: %s",
			len(matches), record.HostName, record.RecordType, strings.Join(values, ", ")))
	}
}

// Ensure creates or updates record so the domain has exactly the record
// described (create-or-update, O4), and reports whether it was created,
// updated, or already matched (unchanged). It is idempotent: running the
// same call twice reports "unchanged" the second time.
//
// Identity rule: for MX and TXT, (hostname, type, value) identifies the
// record, since a hostname can legitimately carry several of each at once;
// Ensure only updates one if its value already matches, otherwise it adds a
// new record alongside the others. For every other type, (hostname, type)
// identifies the record, matching normal DNS practice of a single
// A/AAAA/CNAME/NS per hostname; if more than one already exists, Ensure
// refuses rather than guess which to update (mirrors UpdateRecordMatching).
func (s *Service) Ensure(domainName string, record dnsrecord.Record) (EnsureResult, error) {
	if err := s.ValidateRecord(record); err != nil {
		return EnsureResult{}, fmt.Errorf("invalid record: %w", err)
	}

	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return EnsureResult{}, err
	}

	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return EnsureResult{}, fmt.Errorf("failed to get existing records: %w", err)
	}

	action, idx, err := ensurePlan(existingRecords, record)
	if err != nil {
		return EnsureResult{}, err
	}

	switch action {
	case EnsureStatusUnchanged:
		return EnsureResult{Status: EnsureStatusUnchanged, Record: existingRecords[idx]}, nil

	case EnsureStatusCreated:
		if s.provider.Capabilities().CanCreateRecord {
			created, err := s.provider.CreateRecord(ctx, zoneID, record)
			if err != nil {
				return EnsureResult{}, err
			}
			return EnsureResult{Status: EnsureStatusCreated, Record: created}, nil
		}
		existingRecords = append(existingRecords, record)
		if err := s.provider.BulkReplaceRecords(ctx, zoneID, existingRecords); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Status: EnsureStatusCreated, Record: record}, nil

	case EnsureStatusUpdated:
		recordID := existingRecords[idx].ID
		if s.provider.Capabilities().CanUpdateRecord && recordID != "" {
			updated, err := s.provider.UpdateRecord(ctx, zoneID, recordID, record)
			if err != nil {
				return EnsureResult{}, err
			}
			return EnsureResult{Status: EnsureStatusUpdated, Record: updated}, nil
		}
		existingRecords[idx] = record
		if err := s.provider.BulkReplaceRecords(ctx, zoneID, existingRecords); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Status: EnsureStatusUpdated, Record: record}, nil

	default:
		return EnsureResult{}, fmt.Errorf("unexpected ensure action: %s", action)
	}
}

// PlanEnsure reports what Ensure would do, without writing anything. It
// still calls the provider's read-only ListRecords so the decision
// (created/updated/unchanged) reflects real current state.
func (s *Service) PlanEnsure(domainName string, record dnsrecord.Record) (EnsureResult, error) {
	if err := s.ValidateRecord(record); err != nil {
		return EnsureResult{}, fmt.Errorf("invalid record: %w", err)
	}

	ctx := context.Background()
	zoneID, err := s.resolveZoneID(ctx, domainName)
	if err != nil {
		return EnsureResult{}, err
	}

	existingRecords, err := s.provider.ListRecords(ctx, zoneID)
	if err != nil {
		return EnsureResult{}, fmt.Errorf("failed to get existing records: %w", err)
	}

	action, idx, err := ensurePlan(existingRecords, record)
	if err != nil {
		return EnsureResult{}, err
	}

	if action == EnsureStatusUnchanged {
		return EnsureResult{Status: EnsureStatusUnchanged, Record: existingRecords[idx]}, nil
	}
	return EnsureResult{Status: action, Record: record}, nil
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
