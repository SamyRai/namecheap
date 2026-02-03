package dnsrecord

// Record represents a DNS record
type Record struct {
	ID         string
	HostName   string
	RecordType string
	Address    string
	TTL        int
	MXPref     int

	// Extended fields for Provider Contract v2
	Priority int                    // MX and SRV priority
	Weight   int                    // SRV weight
	Port     int                    // SRV port
	Target   string                 // SRV and MX target
	Metadata map[string]interface{} // Provider-specific metadata
	Raw      interface{}            // Original provider response
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
