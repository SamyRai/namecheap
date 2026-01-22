package dnsrecord

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
