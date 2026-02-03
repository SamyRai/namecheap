package dnsrecord

// Record represents a DNS record
type Record struct {
	ID         string
	HostName   string
	RecordType string
	Address    string
	TTL        int
	MXPref     int
	Priority   int
	Weight     int
	Port       int
	Target     string
	Metadata   map[string]string
	Raw        interface{}
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
