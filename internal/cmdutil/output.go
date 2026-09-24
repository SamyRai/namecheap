package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

// Supported --output values.
const (
	OutputTable = "table"
	OutputJSON  = "json"
	OutputYAML  = "yaml"
)

// ValidOutputFormats lists the accepted --output values, in the order they
// should be presented to users (e.g. in flag help / error messages).
var ValidOutputFormats = []string{OutputTable, OutputJSON, OutputYAML}

// IsValidOutputFormat reports whether format is one zonekit understands.
func IsValidOutputFormat(format string) bool {
	for _, f := range ValidOutputFormats {
		if format == f {
			return true
		}
	}
	return false
}

// WriteResult renders data to w as JSON or YAML. It only handles the
// structured formats: callers own the OutputTable rendering (it varies per
// command) and should branch on format before calling this.
func WriteResult(w io.Writer, format string, data interface{}) error {
	switch format {
	case OutputJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case OutputYAML:
		b, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// commandError is the stable JSON/YAML error envelope written to stderr.
type commandError struct {
	Error string `json:"error" yaml:"error"`
}

// WriteError reports err on w, formatted per format. In table mode it
// matches zonekit's historical "Error: <message>" text; in json/yaml mode it
// emits a single structured object with an "error" field so scripted callers
// can parse failures the same way they parse successful results.
func WriteError(w io.Writer, format string, err error) {
	if err == nil {
		return
	}
	switch format {
	case OutputJSON, OutputYAML:
		_ = WriteResult(w, format, commandError{Error: err.Error()})
	default:
		_, _ = fmt.Fprintf(w, "Error: %v\n", err)
	}
}

// RecordDTO is the stable, snake_case wire shape for a DNS record in
// json/yaml output (O1). Field names are part of the documented contract and
// must not change without a compatible migration.
type RecordDTO struct {
	ID       string `json:"id,omitempty" yaml:"id,omitempty"`
	Hostname string `json:"hostname" yaml:"hostname"`
	Type     string `json:"type" yaml:"type"`
	Value    string `json:"value" yaml:"value"`
	TTL      int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	MXPref   int    `json:"mx_pref,omitempty" yaml:"mx_pref,omitempty"`
}

// NewRecordDTO converts a dnsrecord.Record to its wire representation.
func NewRecordDTO(r dnsrecord.Record) RecordDTO {
	return RecordDTO{
		ID:       r.ID,
		Hostname: r.HostName,
		Type:     r.RecordType,
		Value:    r.Address,
		TTL:      r.TTL,
		MXPref:   r.MXPref,
	}
}

// NewRecordDTOs converts a slice of records to their wire representation.
// It always returns a non-nil, possibly-empty slice so JSON/YAML output is
// "[]"/"[]" rather than "null" when there are no records.
func NewRecordDTOs(records []dnsrecord.Record) []RecordDTO {
	out := make([]RecordDTO, len(records))
	for i, r := range records {
		out[i] = NewRecordDTO(r)
	}
	return out
}
