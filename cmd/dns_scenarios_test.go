package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

// testIPv4 is a placeholder address reused across the fixtures below.
const testIPv4 = "192.168.1.1"

// testHostname is a placeholder hostname reused across the fixtures below.
const testHostname = "www"

// O2: diagnostics ("Using account: ...", "Using config file: ...") must
// never appear on stdout, in any output mode, so stdout stays script-safe.
func TestDiagnostics_NeverOnStdout_TableMode(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: testHostname, RecordType: "A", Address: testIPv4})

	stdout, stderr, err := runCLI(t, mp, "dns", "list", testDomain)
	require.NoError(t, err)
	require.NotContains(t, stdout, "Using account")
	require.Contains(t, stderr, "Using account")
}

func TestDiagnostics_NeverOnStdout_JSONMode(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: testHostname, RecordType: "A", Address: testIPv4})

	stdout, stderr, err := runCLI(t, mp, "--output", "json", "dns", "list", testDomain)
	require.NoError(t, err)
	require.NotContains(t, stdout, "Using account")
	require.Contains(t, stderr, "Using account")

	// stdout must be nothing but the JSON result: parseable as-is.
	var decoded []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &decoded))
}

// O1: dns list --output json/yaml emits stable snake_case fields.
func TestDNSList_JSONOutput_SchemaAndFields(t *testing.T) {
	mp := newTestMockProvider(
		dnsrecord.Record{HostName: "@", RecordType: "MX", Address: "mail.example.com", TTL: 1800, MXPref: 10},
	)

	stdout, _, err := runCLI(t, mp, "--output", "json", "dns", "list", testDomain)
	require.NoError(t, err)

	var records []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &records))
	require.Len(t, records, 1)
	require.Equal(t, "@", records[0]["hostname"])
	require.Equal(t, "MX", records[0]["type"])
	require.Equal(t, "mail.example.com", records[0]["value"])
	require.Equal(t, float64(1800), records[0]["ttl"])
	require.Equal(t, float64(10), records[0]["mx_pref"])
}

func TestDNSList_YAMLOutput_Parses(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: testHostname, RecordType: "A", Address: testIPv4})

	stdout, _, err := runCLI(t, mp, "--output", "yaml", "dns", "list", testDomain)
	require.NoError(t, err)
	require.Contains(t, stdout, "hostname: www")
	require.Contains(t, stdout, "type: A")
}

// O5: --name must be an exact match, not a substring match.
func TestDNSList_NameFilter_ExactNotSubstring(t *testing.T) {
	mp := newTestMockProvider(
		dnsrecord.Record{HostName: testHostname, RecordType: "A", Address: testIPv4},
		dnsrecord.Record{HostName: "www2", RecordType: "A", Address: "192.168.1.2"},
	)

	stdout, _, err := runCLI(t, mp, "--output", "json", "dns", "list", testDomain, "--name", testHostname)
	require.NoError(t, err)

	var records []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &records))
	require.Len(t, records, 1)
	require.Equal(t, testHostname, records[0]["hostname"])
}

// O3: dry-run must never write, on add/update/delete/clear/bulk.

func TestDNSAdd_DryRun_NoWrites(t *testing.T) {
	mp := newTestMockProvider()
	before := recordCount(mp)

	stdout, _, err := runCLI(t, mp, "dns", "add", testDomain, testHostname, "A", testIPv4, "--dry-run")
	require.NoError(t, err)
	require.Equal(t, before, recordCount(mp))
	require.Contains(t, stdout, "DRY RUN")
}

func TestDNSAdd_DryRun_JSONShowsPlannedCreate(t *testing.T) {
	mp := newTestMockProvider()

	stdout, _, err := runCLI(t, mp, "--output", "json", "dns", "add", testDomain, testHostname, "A", testIPv4, "--dry-run")
	require.NoError(t, err)
	require.Equal(t, 0, recordCount(mp))

	var plans []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &plans))
	require.Len(t, plans, 1)
	require.Equal(t, "create", plans[0]["action"])
	after := plans[0]["after"].(map[string]interface{})
	require.Equal(t, testHostname, after["hostname"])
}

func TestDNSUpdate_DryRun_NoWrites(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: "@", RecordType: "A", Address: testIPv4})

	_, _, err := runCLI(t, mp, "dns", "update", testDomain, "@", "A", "192.168.1.100", "--dry-run")
	require.NoError(t, err)

	records := mp.Records[testDomain]
	require.Len(t, records, 1)
	for _, r := range records {
		require.Equal(t, testIPv4, r.Address, "dry-run must not change the stored value")
	}
}

func TestDNSDelete_DryRun_NoWrites(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: "@", RecordType: "A", Address: testIPv4})

	_, _, err := runCLI(t, mp, "dns", "delete", testDomain, "@", "A", "--dry-run")
	require.NoError(t, err)
	require.Equal(t, 1, recordCount(mp))
}

func TestDNSBulk_DryRun_NoWrites(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: "@", RecordType: "A", Address: testIPv4})
	opsFile := writeBulkOpsFile(t)

	_, _, err := runCLI(t, mp, "dns", "bulk", testDomain, opsFile, "--dry-run")
	require.NoError(t, err)
	require.Equal(t, 1, recordCount(mp), "dry-run bulk must not apply the add operation")
}

// O3: dns clear requires --confirm, even with no other flags, and --dry-run
// previews without requiring --confirm at all.
func TestDNSClear_WithoutConfirm_Refuses(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: "@", RecordType: "A", Address: testIPv4})

	_, _, err := runCLI(t, mp, "dns", "clear", testDomain)
	require.Error(t, err)
	require.Equal(t, 1, recordCount(mp), "refused clear must not delete anything")
}

func TestDNSClear_DryRun_NoConfirmNeeded_NoWrites(t *testing.T) {
	mp := newTestMockProvider(
		dnsrecord.Record{HostName: "@", RecordType: "A", Address: testIPv4},
		dnsrecord.Record{HostName: testHostname, RecordType: "A", Address: "192.168.1.2"},
	)

	stdout, _, err := runCLI(t, mp, "dns", "clear", testDomain, "--dry-run")
	require.NoError(t, err)
	require.Equal(t, 2, recordCount(mp))
	require.Contains(t, stdout, "DRY RUN")
}

func TestDNSClear_WithConfirm_Clears(t *testing.T) {
	mp := newTestMockProvider(dnsrecord.Record{HostName: "@", RecordType: "A", Address: testIPv4})

	_, _, err := runCLI(t, mp, "dns", "clear", testDomain, "--confirm")
	require.NoError(t, err)
	require.Equal(t, 0, recordCount(mp))
}

// O4: dns ensure is idempotent end-to-end.
func TestDNSEnsure_IdempotentAcrossTwoRuns(t *testing.T) {
	mp := newTestMockProvider()

	stdout1, _, err := runCLI(t, mp, "--output", "json", "dns", "ensure", testDomain, testHostname, "A", testIPv4)
	require.NoError(t, err)
	var result1 map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout1), &result1))
	require.Equal(t, "created", result1["status"])
	require.Equal(t, 1, recordCount(mp))

	stdout2, _, err := runCLI(t, mp, "--output", "json", "dns", "ensure", testDomain, testHostname, "A", testIPv4)
	require.NoError(t, err)
	var result2 map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout2), &result2))
	require.Equal(t, "unchanged", result2["status"])
	require.Equal(t, 1, recordCount(mp), "a repeated ensure must not create a duplicate")
}

func TestDNSEnsure_DryRun_NoWrites(t *testing.T) {
	mp := newTestMockProvider()

	stdout, _, err := runCLI(t, mp, "--output", "json", "dns", "ensure", testDomain, testHostname, "A", testIPv4, "--dry-run")
	require.NoError(t, err)
	require.Equal(t, 0, recordCount(mp))

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, "created", result["status"])
}

// Errors in json/yaml mode must go to stderr and exit non-zero.
func TestErrors_JSONMode_GoToStderrOnly(t *testing.T) {
	mp := newTestMockProvider()

	stdout, stderr, err := runCLI(t, mp, "--output", "json", "dns", "list", "not a valid domain")
	require.Error(t, err)
	require.Empty(t, stdout, "an error must never contaminate stdout")
	require.NotEmpty(t, stderr)

	// Verify it's a parseable structured error, not free text. WriteError
	// pretty-prints (multi-line), and stderr may also carry unrelated
	// diagnostics (e.g. "Using config file: ..."), so isolate the trailing
	// JSON object rather than assuming it's a single line.
	start := strings.LastIndex(stderr, "{")
	require.GreaterOrEqual(t, start, 0, "expected a JSON error object on stderr, got: %q", stderr)
	var decoded map[string]string
	require.NoError(t, json.Unmarshal([]byte(stderr[start:]), &decoded))
	require.NotEmpty(t, decoded["error"])
}

// --- helpers -------------------------------------------------------------

// writeBulkOpsFile writes a bulk operations file in the shape
// parseBulkOperationsFile (cmd/dns.go) actually parses: a top-level list, not
// the "operations:"-wrapped map shown in dnsBulkCmd's Long help text. That
// mismatch between the documented example and the parser is pre-existing and
// out of this change's scope; noted for a follow-up doc/parser fix.
func writeBulkOpsFile(t *testing.T) string {
	t.Helper()
	content := `- action: add
  hostname: www
  type: A
  value: 192.168.1.2
`
	path := filepath.Join(t.TempDir(), "bulk-ops.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
