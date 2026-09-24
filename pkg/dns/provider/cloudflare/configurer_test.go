package cloudflare

import (
	"context"
	"testing"

	dnsprovider "go.glpx.pro/zonekit/pkg/dns/provider"

	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestZoneSettings_GetFiltersToManagedIDs(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	fs.settings["zone-1"] = map[string]interface{}{SettingSSL: "strict"}

	p := newTestProvider(t, fs)
	settings, err := p.GetZoneSettings(context.Background(), "zone-1")
	require.NoError(t, err)

	require.Equal(t, "strict", settings[SettingSSL])
	_, hasBrotli := settings["brotli"]
	require.False(t, hasBrotli, "an unmanaged setting must not appear in GetZoneSettings")
}

func TestZoneSettings_SetSendsOnlyGivenItems(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	p := newTestProvider(t, fs)

	err := p.SetZoneSettings(context.Background(), "zone-1", dnsprovider.ZoneSettings{
		SettingSSL:            "strict",
		SettingMinTLSVersion:  "1.2",
		SettingTLS13:          "on",
		SettingAlwaysUseHTTPS: "on",
	})
	require.NoError(t, err)

	fs.mu.Lock()
	got := fs.settings["zone-1"]
	fs.mu.Unlock()

	require.Equal(t, "strict", got[SettingSSL])
	require.Equal(t, "1.2", got[SettingMinTLSVersion])
	require.Equal(t, "on", got[SettingTLS13])
	require.Equal(t, "on", got[SettingAlwaysUseHTTPS])
	// automatic_https_rewrites was never set - it must be absent, not zeroed.
	_, ok := got[SettingAutomaticHTTPSRewrites]
	require.False(t, ok)
}

// TestBotManagement_PreservesUnknownFields is the read-modify-write
// contract: SetBotManagement must not clobber fields Cloudflare exposes
// that dnsprovider.BotManagement does not model.
func TestBotManagement_PreservesUnknownFields(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	fs.botMgmt["zone-1"] = map[string]interface{}{
		"fight_mode":                false,
		"crawler_protection":        false,
		"enable_js":                 false,
		"sbfm_definitely_automated": "block", // unmodeled field
		"ai_bots_protection":        "block", // unmodeled field
	}

	p := newTestProvider(t, fs)
	err := p.SetBotManagement(context.Background(), "zone-1", dnsprovider.BotManagement{
		FightMode: boolPtr(true),
	})
	require.NoError(t, err)

	fs.mu.Lock()
	raw := fs.botMgmt["zone-1"]
	fs.mu.Unlock()

	require.Equal(t, true, raw["fight_mode"], "the explicitly requested field must change")
	require.Equal(t, false, raw["crawler_protection"], "fields not passed must be left as-is")
	require.Equal(t, "block", raw["sbfm_definitely_automated"], "unmodeled fields must survive the read-modify-write")
	require.Equal(t, "block", raw["ai_bots_protection"], "unmodeled fields must survive the read-modify-write")

	got, err := p.GetBotManagement(context.Background(), "zone-1")
	require.NoError(t, err)
	require.NotNil(t, got.FightMode)
	require.True(t, *got.FightMode)
}

func TestSecurityTXT_RoundTrip(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	p := newTestProvider(t, fs)
	ctx := context.Background()

	want := dnsprovider.SecurityTXT{
		Enabled:            true,
		Contact:            []string{"mailto:security@example.com"},
		Expires:            "2027-01-01T00:00:00.000Z",
		PreferredLanguages: []string{"en", "de"},
	}
	require.NoError(t, p.SetSecurityTXT(ctx, "zone-1", want))

	got, err := p.GetSecurityTXT(ctx, "zone-1")
	require.NoError(t, err)
	require.Equal(t, want.Contact, got.Contact)
	require.Equal(t, want.PreferredLanguages, got.PreferredLanguages)
	require.True(t, got.Enabled)
}

// TestRedirectRules_MergePreservesForeignRule is the key ruleset-merge
// contract: a rule with no ref matching our prefix must survive
// SetRedirectRules byte-for-byte, in position, even though we replace
// every rule that IS ours.
func TestRedirectRules_MergePreservesForeignRule(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")

	foreignRule := map[string]interface{}{
		redirectFieldRef:         "someone-elses-tool-abc123",
		redirectFieldDescription: "hand-authored redirect",
		redirectFieldExpression:  `http.request.uri.path eq "/legacy"`,
		redirectFieldEnabled:     true,
		redirectFieldAction:      redirectActionRedirect,
		"action_parameters": map[string]interface{}{
			"from_value": map[string]interface{}{
				"status_code": float64(301),
				"target_url": map[string]interface{}{
					redirectFieldExpression: `concat("https://example.com/new")`,
				},
			},
		},
	}
	ourOldRule := map[string]interface{}{
		redirectFieldRef:        "zonekit-abc",
		redirectFieldExpression: `http.request.uri.path eq "/old"`,
		redirectFieldAction:     redirectActionRedirect,
	}
	fs.redirects["zone-1"] = rulesetWire{Rules: []map[string]interface{}{foreignRule, ourOldRule}}

	p := newTestProvider(t, fs)
	ctx := context.Background()

	err := p.SetRedirectRules(ctx, "zone-1", "zonekit-", []dnsprovider.RedirectRule{
		{Ref: "zonekit-new", Expression: `http.request.uri.path eq "/new-path"`, TargetExpression: `concat("https://example.com/target")`, StatusCode: 302},
	})
	require.NoError(t, err)

	fs.mu.Lock()
	rules := fs.redirects["zone-1"].Rules
	fs.mu.Unlock()
	require.Len(t, rules, 2, "the foreign rule plus our one replacement rule")

	require.Equal(t, foreignRule, rules[0], "the foreign rule must be byte-for-byte unchanged")

	ours, err := p.GetRedirectRules(ctx, "zone-1", "zonekit-")
	require.NoError(t, err)
	require.Len(t, ours, 1)
	require.Equal(t, "zonekit-new", ours[0].Ref)
	require.Equal(t, 302, ours[0].StatusCode)

	foreign, err := p.GetRedirectRules(ctx, "zone-1", "someone-elses-tool-")
	require.NoError(t, err)
	require.Len(t, foreign, 1)
	require.Equal(t, "someone-elses-tool-abc123", foreign[0].Ref)
}

func TestSetRedirectRules_RequiresRefPrefix(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	p := newTestProvider(t, fs)

	err := p.SetRedirectRules(context.Background(), "zone-1", "", []dnsprovider.RedirectRule{{Ref: "x"}})
	require.Error(t, err)
}

func TestDNSSEC_GetAndSet(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	fs.dnssec["zone-1"] = dnssecWire{Status: "disabled"}

	p := newTestProvider(t, fs)
	ctx := context.Background()

	status, err := p.GetDNSSEC(ctx, "zone-1")
	require.NoError(t, err)
	require.Equal(t, "disabled", status.Status)

	enabled, err := p.SetDNSSEC(ctx, "zone-1", true)
	require.NoError(t, err)
	require.Equal(t, "active", enabled.Status)
	require.NotEmpty(t, enabled.DS)
}
