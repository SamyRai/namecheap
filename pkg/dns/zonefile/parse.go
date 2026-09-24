package zonefile

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

// Parse reads a zone file previously produced by Format and returns the DNS
// records it describes.
//
// Parse only understands the subset of master-file syntax Format emits: one
// record per line, with an explicit owner, TTL and class on every line, and
// "; zonekit:url-redirect" comments for Namecheap's URL/URL301/FRAME
// pseudo-records (see Format's doc comment for why those aren't real
// records). $ORIGIN/other $-directives and blank/plain-comment lines are
// skipped. Multi-line or parenthesised constructs (e.g. a hand-written SOA
// block) are not supported -- Format never emits one, and zonekit has no
// use for one on import.
func Parse(content string) ([]dnsrecord.Record, error) {
	var records []dnsrecord.Record

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		trimmed := strings.TrimSpace(scanner.Text())

		record, skip, err := parseLine(trimmed)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if skip {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read zone file: %w", err)
	}

	return records, nil
}

func parseLine(trimmed string) (record dnsrecord.Record, skip bool, err error) {
	switch {
	case trimmed == "", strings.HasPrefix(trimmed, "$"):
		return dnsrecord.Record{}, true, nil
	case strings.HasPrefix(trimmed, urlRedirectPrefix):
		record, err = parseURLRedirectComment(trimmed)
		return record, false, err
	case strings.HasPrefix(trimmed, ";"):
		return dnsrecord.Record{}, true, nil
	default:
		record, err = parseRecordLine(trimmed)
		return record, false, err
	}
}

// parseRecordLine parses one "<owner> <ttl> IN <type> <rdata...>" line.
func parseRecordLine(line string) (dnsrecord.Record, error) {
	tokens := tokenize(line)
	if len(tokens) < 4 {
		return dnsrecord.Record{}, fmt.Errorf("malformed record line: %q", line)
	}

	owner, ttlToken, class, typeToken := tokens[0], tokens[1], tokens[2], tokens[3]

	ttl, err := strconv.Atoi(ttlToken)
	if err != nil {
		return dnsrecord.Record{}, fmt.Errorf("invalid TTL %q: %w", ttlToken, err)
	}
	if !strings.EqualFold(class, "IN") {
		return dnsrecord.Record{}, fmt.Errorf("unsupported record class %q (only IN is supported)", class)
	}

	record := dnsrecord.Record{
		HostName:   owner,
		RecordType: strings.ToUpper(typeToken),
		TTL:        ttl,
	}

	rest := tokens[4:]
	if err := fillRecordData(&record, rest, line); err != nil {
		return dnsrecord.Record{}, err
	}

	return record, nil
}

func fillRecordData(record *dnsrecord.Record, rest []string, line string) error {
	switch record.RecordType {
	case dnsrecord.RecordTypeTXT:
		value, err := unquoteTXT(rest)
		if err != nil {
			return err
		}
		record.Address = value
	case dnsrecord.RecordTypeMX:
		if len(rest) != 2 {
			return fmt.Errorf("malformed MX record: %q", line)
		}
		pref, err := strconv.Atoi(rest[0])
		if err != nil {
			return fmt.Errorf("invalid MX preference %q: %w", rest[0], err)
		}
		record.MXPref = pref
		record.Address = rest[1]
	default:
		if len(rest) == 0 {
			return fmt.Errorf("record %s %s is missing a value: %q", record.HostName, record.RecordType, line)
		}
		record.Address = strings.Join(rest, " ")
	}
	return nil
}

// parseURLRedirectComment parses a "; zonekit:url-redirect <owner> <type>
// "<value>"" comment back into the pseudo-record it encodes.
func parseURLRedirectComment(trimmed string) (dnsrecord.Record, error) {
	rest := strings.TrimPrefix(trimmed, urlRedirectPrefix)
	tokens := tokenize(rest)
	if len(tokens) != 3 {
		return dnsrecord.Record{}, fmt.Errorf("malformed url-redirect comment: %q", trimmed)
	}

	owner, recordType, quotedValue := tokens[0], strings.ToUpper(tokens[1]), tokens[2]
	if !isPseudoRedirectType(recordType) {
		return dnsrecord.Record{}, fmt.Errorf("unknown url-redirect type %q", recordType)
	}

	value, err := unquoteTXT([]string{quotedValue})
	if err != nil {
		return dnsrecord.Record{}, err
	}

	return dnsrecord.Record{
		HostName:   owner,
		RecordType: recordType,
		Address:    value,
	}, nil
}

// unquoteTXT reverses quoteTXT: it strips the surrounding quotes and
// backslash-escaping from each <character-string> chunk and concatenates
// them back into the original raw value.
func unquoteTXT(chunks []string) (string, error) {
	if len(chunks) == 0 {
		return "", fmt.Errorf("expected at least one quoted character-string, got none")
	}

	var sb strings.Builder
	for _, chunk := range chunks {
		if len(chunk) < 2 || chunk[0] != '"' || chunk[len(chunk)-1] != '"' {
			return "", fmt.Errorf("malformed character-string (expected quotes): %q", chunk)
		}
		sb.WriteString(unescapeTXT(chunk[1 : len(chunk)-1]))
	}
	return sb.String(), nil
}

func unescapeTXT(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	escaped := false
	for _, r := range s {
		if escaped {
			sb.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// tokenize splits line on whitespace, treating a double-quoted substring
// (with \" and \\ escapes honored) as a single token that keeps its
// surrounding quotes -- so a TXT character-string's internal spaces don't
// get split into separate tokens.
func tokenize(line string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	escaped := false

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}

	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case inQuotes && r == '\\':
			cur.WriteRune(r)
			escaped = true
		case r == '"':
			cur.WriteRune(r)
			inQuotes = !inQuotes
		case !inQuotes && (r == ' ' || r == '\t'):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()

	return tokens
}
