package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"go.glpx.pro/zonekit/pkg/client"
	"go.glpx.pro/zonekit/pkg/dns"
	"go.glpx.pro/zonekit/pkg/dns/provider"
	"go.glpx.pro/zonekit/pkg/dns/provider/conformance"
	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

// --- test scaffolding --------------------------------------------------

// resetAllFlags restores every flag under cmd (and its subtree) to its
// default value and clears Changed. The dns/domain/account/root commands are
// package-level singletons reused across every test in this package (and by
// the real CLI), so a flag set by one test (e.g. --confirm) would otherwise
// leak into the next.
func resetAllFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VisitAll(resetFlag)
	cmd.Flags().VisitAll(resetFlag)
	for _, c := range cmd.Commands() {
		resetAllFlags(c)
	}
}

func resetFlag(f *pflag.Flag) {
	_ = f.Value.Set(f.DefValue)
	f.Changed = false
}

// writeTestConfig writes a minimal, valid zonekit account config to a temp
// file and returns its path. The credentials are fake but well-formed:
// client.NewClient only checks they're non-empty, it never makes a network
// call by itself.
func writeTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "zonekit-test-config.yaml")
	content := `current_account: test
accounts:
  test:
    username: testuser
    api_user: testuser
    api_key: testkey
    client_ip: 127.0.0.1
    use_sandbox: true
    description: test account
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

const testDomain = "example.com"

// newTestMockProvider returns a conformance.MockProvider pre-seeded with a
// zone for testDomain, optionally pre-loaded with records.
func newTestMockProvider(records ...dnsrecord.Record) *conformance.MockProvider {
	mp := conformance.NewMockProvider()
	mp.Zones[testDomain] = provider.Zone{ID: testDomain, Name: testDomain}
	mp.Records[testDomain] = make(map[string]dnsrecord.Record)
	for i, r := range records {
		if r.ID == "" {
			r.ID = "rec-" + string(rune('a'+i))
		}
		mp.Records[testDomain][r.ID] = r
	}
	return mp
}

// recordCount returns how many records the mock currently holds for
// testDomain, for asserting a dry-run made zero writes.
func recordCount(mp *conformance.MockProvider) int {
	return len(mp.Records[testDomain])
}

// captureOutput redirects os.Stdout/os.Stderr for the duration of fn and
// returns everything written to each. zonekit's command bodies print
// directly to os.Stdout/os.Stderr (not cmd.OutOrStdout()), so this is a
// process-level capture rather than a cobra-level one.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	errR, errW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = outW
	os.Stderr = errW

	outCh := make(chan string, 1)
	errCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, outR)
		outCh <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, errR)
		errCh <- buf.String()
	}()

	func() {
		defer func() {
			os.Stdout = origOut
			os.Stderr = origErr
			_ = outW.Close()
			_ = errW.Close()
		}()
		fn()
	}()

	return <-outCh, <-errCh
}

// runCLI resets global CLI state, points it at a throwaway config file and a
// mock-backed DNS service, executes rootCmd with args, and returns the
// captured stdout/stderr and any error. No real provider or network is ever
// touched.
func runCLI(t *testing.T, mp *conformance.MockProvider, args ...string) (stdout, stderr string, runErr error) {
	t.Helper()

	resetAllFlags(rootCmd)
	cfgFile = writeTestConfig(t)
	accountName = ""
	outputFormat = "table"

	origFactory := newDNSService
	t.Cleanup(func() { newDNSService = origFactory })
	newDNSService = func(_ *client.Client) *dns.Service {
		return dns.NewServiceWithProvider(mp)
	}

	rootCmd.SetArgs(args)
	stdout, stderr = captureOutput(t, func() {
		// Execute (not rootCmd.Execute directly) is what main() calls: it
		// silences cobra's own error/usage printing and writes the error in
		// the format --output asked for. Testing through it, not around it,
		// is the point of the JSON-error-on-stderr assertions below.
		runErr = Execute()
	})
	return stdout, stderr, runErr
}
