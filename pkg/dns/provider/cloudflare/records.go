package cloudflare

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/errors"
)

// maxCommentLength is Cloudflare's Free-plan limit on a DNS record
// comment. Tags are a paid-plan feature and are deliberately not
// supported here.
const maxCommentLength = 100

// Metadata keys used to round-trip fields dnsrecord.Record has no typed
// slot for. Kept as plain strings (rather than typed booleans/ints) since
// Record.Metadata is map[string]string.
const (
	metaProxied = "proxied"
	metaComment = "comment"
	metaFlags   = "flags" // CAA
	metaTag     = "tag"   // CAA
)

// dnsRecordWire is the Cloudflare DNS record wire shape. Content is used
// by simple record types (A/AAAA/CNAME/MX/TXT/NS); SRV and CAA use the
// structured Data object instead, matching Cloudflare's API.
type dnsRecordWire struct {
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Content  string          `json:"content,omitempty"`
	TTL      int             `json:"ttl,omitempty"`
	Priority *int            `json:"priority,omitempty"`
	Proxied  *bool           `json:"proxied,omitempty"`
	Comment  string          `json:"comment,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type srvData struct {
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Port     int    `json:"port"`
	Target   string `json:"target"`
}

type caaData struct {
	Flags int    `json:"flags"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// toWire converts a zonekit record into the Cloudflare wire shape,
// validating fields Cloudflare's Free plan constrains (comment length).
func toWire(r dnsrecord.Record) (dnsRecordWire, error) {
	if len(r.Metadata[metaComment]) > maxCommentLength {
		return dnsRecordWire{}, errors.NewInvalidInput("comment",
			fmt.Sprintf("must be at most %d characters on the Free plan (got %d)", maxCommentLength, len(r.Metadata[metaComment])))
	}

	w := dnsRecordWire{
		ID:      r.ID,
		Type:    strings.ToUpper(r.RecordType),
		Name:    r.HostName,
		TTL:     r.TTL,
		Comment: r.Metadata[metaComment],
	}
	if w.TTL == 0 {
		w.TTL = 1 // Cloudflare's "automatic" TTL
	}

	if proxied, ok := r.Metadata[metaProxied]; ok {
		v := proxied == "true"
		w.Proxied = &v
	}

	switch w.Type {
	case dnsrecord.RecordTypeSRV:
		data, err := json.Marshal(srvData{Priority: r.Priority, Weight: r.Weight, Port: r.Port, Target: r.Target})
		if err != nil {
			return dnsRecordWire{}, fmt.Errorf("cloudflare: encode SRV data: %w", err)
		}
		w.Data = data
	case dnsrecord.RecordTypeCAA:
		flags, _ := strconv.Atoi(r.Metadata[metaFlags])
		data, err := json.Marshal(caaData{Flags: flags, Tag: r.Metadata[metaTag], Value: r.Address})
		if err != nil {
			return dnsRecordWire{}, fmt.Errorf("cloudflare: encode CAA data: %w", err)
		}
		w.Data = data
	case dnsrecord.RecordTypeMX:
		w.Content = r.Address
		pref := r.MXPref
		w.Priority = &pref
	default:
		w.Content = r.Address
	}

	return w, nil
}

// fromWire converts a Cloudflare wire record back into a zonekit record.
func fromWire(w dnsRecordWire) dnsrecord.Record {
	r := dnsrecord.Record{
		ID:         w.ID,
		HostName:   w.Name,
		RecordType: strings.ToUpper(w.Type),
		TTL:        w.TTL,
		Metadata:   map[string]string{},
		Raw:        w,
	}

	if w.Comment != "" {
		r.Metadata[metaComment] = w.Comment
	}
	if w.Proxied != nil {
		r.Metadata[metaProxied] = strconv.FormatBool(*w.Proxied)
	}

	switch r.RecordType {
	case dnsrecord.RecordTypeSRV:
		var d srvData
		_ = json.Unmarshal(w.Data, &d)
		r.Priority, r.Weight, r.Port, r.Target = d.Priority, d.Weight, d.Port, d.Target
	case dnsrecord.RecordTypeCAA:
		var d caaData
		_ = json.Unmarshal(w.Data, &d)
		r.Metadata[metaFlags] = strconv.Itoa(d.Flags)
		r.Metadata[metaTag] = d.Tag
		r.Address = d.Value
	case dnsrecord.RecordTypeMX:
		r.Address = w.Content
		if w.Priority != nil {
			r.MXPref = *w.Priority
		}
	case dnsrecord.RecordTypeTXT:
		r.Address = canonicalizeTXT(w.Content)
	default:
		r.Address = w.Content
	}

	return r
}

// canonicalizeTXT normalizes a TXT record's presentation so that
// comparisons between what we sent and what Cloudflare returns (or
// between two independently-built record sets) do not spuriously differ.
// Cloudflare may return a single logical value as multiple
// double-quoted, space-separated segments (matching DNS zonefile TXT
// chunking for values over 255 bytes) - e.g. `"v=spf1 " "include:x ~all"`
// - so this strips the quoting and rejoins the segments verbatim.
func canonicalizeTXT(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, `"`) {
		return trimmed
	}

	var b strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range trimmed {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\' && inQuotes:
			escaped = true
		case r == '"':
			inQuotes = !inQuotes
		case inQuotes:
			b.WriteRune(r)
		default:
			// Outside quotes - e.g. the space separating two adjacent
			// quoted segments - is a presentation artifact, not part of
			// the value, so it is dropped rather than joined in.
		}
	}
	return b.String()
}

// recordIdentity returns a stable key identifying "the same logical
// record" across a diff, independent of mutable presentation fields
// (TTL, proxied, comment). Two records sharing an identity are the same
// record with possibly-changed metadata; distinct identities are
// distinct records, since (hostname, type) alone is not unique for
// TXT/MX/SRV/CAA.
func recordIdentity(r dnsrecord.Record) string {
	host := strings.ToLower(strings.TrimSuffix(r.HostName, "."))
	rtype := strings.ToUpper(r.RecordType)

	var value string
	switch rtype {
	case dnsrecord.RecordTypeTXT:
		value = canonicalizeTXT(r.Address)
	case dnsrecord.RecordTypeSRV:
		value = fmt.Sprintf("%d:%d:%d:%s", r.Priority, r.Weight, r.Port, normalizeName(r.Target))
	case dnsrecord.RecordTypeCAA:
		value = fmt.Sprintf("%s:%s:%s", r.Metadata[metaFlags], r.Metadata[metaTag], r.Address)
	case dnsrecord.RecordTypeMX:
		value = fmt.Sprintf("%d:%s", r.MXPref, normalizeName(r.Address))
	default:
		value = strings.ToLower(r.Address)
	}

	return rtype + "|" + host + "|" + value
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSuffix(s, "."))
}

// needsMetadataUpdate reports whether presentation fields differ between
// an existing record and the desired one, even though they share an
// identity (so the underlying value is unchanged).
func needsMetadataUpdate(existing, desired dnsrecord.Record) bool {
	existingTTL, desiredTTL := existing.TTL, desired.TTL
	if existingTTL == 0 {
		existingTTL = 1
	}
	if desiredTTL == 0 {
		desiredTTL = 1
	}
	if existingTTL != desiredTTL {
		return true
	}
	if existing.Metadata[metaProxied] != desired.Metadata[metaProxied] {
		return true
	}
	if existing.Metadata[metaComment] != desired.Metadata[metaComment] {
		return true
	}
	return false
}

// recordUpdate pairs a desired record with the provider-assigned ID of
// the existing record it replaces.
type recordUpdate struct {
	id     string
	record dnsrecord.Record
}

// recordDiff is the set of create/update/delete operations needed to
// turn an existing record set into a desired one.
type recordDiff struct {
	toCreate []dnsrecord.Record
	toUpdate []recordUpdate
	toDelete []dnsrecord.Record
}

// diffRecords computes the minimal set of operations to reconcile
// existing into desired, so BulkReplaceRecords never has to delete
// everything and re-create it - see Provider.BulkReplaceRecords.
func diffRecords(existing, desired []dnsrecord.Record) recordDiff {
	existingByKey := make(map[string]dnsrecord.Record, len(existing))
	for _, r := range existing {
		existingByKey[recordIdentity(r)] = r
	}

	var diff recordDiff
	seen := make(map[string]bool, len(desired))
	for _, d := range desired {
		key := recordIdentity(d)
		seen[key] = true

		ex, ok := existingByKey[key]
		if !ok {
			diff.toCreate = append(diff.toCreate, d)
			continue
		}
		if needsMetadataUpdate(ex, d) {
			diff.toUpdate = append(diff.toUpdate, recordUpdate{id: ex.ID, record: d})
		}
	}

	for _, ex := range existing {
		if !seen[recordIdentity(ex)] {
			diff.toDelete = append(diff.toDelete, ex)
		}
	}

	return diff
}
