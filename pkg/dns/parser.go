package dns

import (
	"fmt"
	"io"
	"os"
	"strings"

	"zonekit/pkg/dnsrecord"

	"github.com/miekg/dns"
)

// ParseZoneFile reads a standard BIND zone file and returns a slice of Records
func ParseZoneFile(filePath string, origin string) ([]dnsrecord.Record, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zone file: %w", err)
	}
	defer file.Close()

	return ParseZone(file, origin, filePath)
}

// ParseZone parses a zone from an io.Reader
func ParseZone(r io.Reader, origin string, file string) ([]dnsrecord.Record, error) {
	if !strings.HasSuffix(origin, ".") {
		origin += "."
	}

	parser := dns.NewZoneParser(r, origin, file)
	var records []dnsrecord.Record

	for rr, ok := parser.Next(); ok; rr, ok = parser.Next() {
		record := convertRRToRecord(rr, origin)
		if record != nil {
			records = append(records, *record)
		}
	}

	if err := parser.Err(); err != nil {
		return nil, fmt.Errorf("error parsing zone file: %w", err)
	}

	return records, nil
}

func convertRRToRecord(rr dns.RR, origin string) *dnsrecord.Record {
	header := rr.Header()
	
	// Clean up hostname
	hostname := strings.TrimSuffix(header.Name, ".")
	originTrimmed := strings.TrimSuffix(origin, ".")
	
	if hostname == originTrimmed {
		hostname = "@"
	} else if strings.HasSuffix(hostname, "."+originTrimmed) {
		hostname = strings.TrimSuffix(hostname, "."+originTrimmed)
	}

	record := &dnsrecord.Record{
		HostName: hostname,
		TTL:      int(header.Ttl),
	}

	switch v := rr.(type) {
	case *dns.A:
		record.RecordType = dnsrecord.RecordTypeA
		record.Address = v.A.String()
	case *dns.AAAA:
		record.RecordType = dnsrecord.RecordTypeAAAA
		record.Address = v.AAAA.String()
	case *dns.CNAME:
		record.RecordType = dnsrecord.RecordTypeCNAME
		record.Address = strings.TrimSuffix(v.Target, ".")
	case *dns.MX:
		record.RecordType = dnsrecord.RecordTypeMX
		record.Address = strings.TrimSuffix(v.Mx, ".")
		record.MXPref = int(v.Preference)
		record.Priority = int(v.Preference)
		record.Target = record.Address
	case *dns.TXT:
		record.RecordType = dnsrecord.RecordTypeTXT
		record.Address = strings.Join(v.Txt, " ")
	case *dns.NS:
		record.RecordType = dnsrecord.RecordTypeNS
		record.Address = strings.TrimSuffix(v.Ns, ".")
	case *dns.SRV:
		record.RecordType = dnsrecord.RecordTypeSRV
		record.Priority = int(v.Priority)
		record.Weight = int(v.Weight)
		record.Port = int(v.Port)
		record.Target = strings.TrimSuffix(v.Target, ".")
		record.Address = fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, record.Target)
	default:
		// Unsupported or ignored record type (like SOA)
		return nil
	}

	return record
}
