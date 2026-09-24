package cmdutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

func TestIsValidOutputFormat(t *testing.T) {
	require.True(t, IsValidOutputFormat(OutputTable))
	require.True(t, IsValidOutputFormat(OutputJSON))
	require.True(t, IsValidOutputFormat(OutputYAML))
	require.False(t, IsValidOutputFormat("xml"))
	require.False(t, IsValidOutputFormat(""))
}

func TestNewRecordDTO_SnakeCaseFields(t *testing.T) {
	r := dnsrecord.Record{
		ID:         "rec-1",
		HostName:   "www",
		RecordType: "A",
		Address:    "192.168.1.1",
		TTL:        1800,
		MXPref:     0,
	}

	dto := NewRecordDTO(r)
	require.Equal(t, "rec-1", dto.ID)
	require.Equal(t, "www", dto.Hostname)
	require.Equal(t, "A", dto.Type)
	require.Equal(t, "192.168.1.1", dto.Value)
	require.Equal(t, 1800, dto.TTL)

	var buf bytes.Buffer
	require.NoError(t, WriteResult(&buf, OutputJSON, dto))

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.Equal(t, "rec-1", decoded["id"])
	require.Equal(t, "www", decoded["hostname"])
	require.Equal(t, "A", decoded["type"])
	require.Equal(t, "192.168.1.1", decoded["value"])
	require.Equal(t, float64(1800), decoded["ttl"])
	// mx_pref is omitempty and zero here, so it must not appear at all.
	_, hasMXPref := decoded["mx_pref"]
	require.False(t, hasMXPref)
}

func TestNewRecordDTOs_EmptyIsEmptyArrayNotNull(t *testing.T) {
	dtos := NewRecordDTOs(nil)
	require.NotNil(t, dtos)
	require.Len(t, dtos, 0)

	var buf bytes.Buffer
	require.NoError(t, WriteResult(&buf, OutputJSON, dtos))
	require.Equal(t, "[]\n", buf.String())
}

func TestWriteResult_JSONParsesBackToList(t *testing.T) {
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: "MX", Address: "mail.example.com", TTL: 1800, MXPref: 10},
		{HostName: "www", RecordType: "A", Address: "192.168.1.1"},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteResult(&buf, OutputJSON, NewRecordDTOs(records)))

	var decoded []RecordDTO
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.Len(t, decoded, 2)
	require.Equal(t, "mail.example.com", decoded[0].Value)
	require.Equal(t, 10, decoded[0].MXPref)
	require.Equal(t, "www", decoded[1].Hostname)
}

func TestWriteResult_YAMLParsesBackToList(t *testing.T) {
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: "TXT", Address: "v=spf1 -all"},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteResult(&buf, OutputYAML, NewRecordDTOs(records)))

	var decoded []RecordDTO
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))
	require.Len(t, decoded, 1)
	require.Equal(t, "v=spf1 -all", decoded[0].Value)
}

func TestWriteResult_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := WriteResult(&buf, "xml", RecordDTO{})
	require.Error(t, err)
}

func TestWriteError_TableModeIsPlainText(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, OutputTable, errors.New("boom"))
	require.Equal(t, "Error: boom\n", buf.String())
}

func TestWriteError_JSONModeIsStructured(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, OutputJSON, errors.New("boom"))

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.Equal(t, "boom", decoded["error"])
}

func TestWriteError_YAMLModeIsStructured(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, OutputYAML, errors.New("boom"))

	var decoded map[string]string
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))
	require.Equal(t, "boom", decoded["error"])
}

func TestWriteError_NilErrorWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, OutputJSON, nil)
	require.Equal(t, 0, buf.Len())
}
