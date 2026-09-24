package provider

import "context"

// ZoneConfigurer is an optional capability for providers that can manage
// zone-level configuration beyond DNS records: SSL/TLS settings, bot
// management, security.txt, redirect rules, and DNSSEC.
//
// It is deliberately NOT part of the Provider interface: most providers
// (Namecheap included) have no equivalent of these zone-level controls.
// A caller discovers support in two steps:
//
//  1. Check the relevant ProviderCapabilities flag (e.g. CanConfigureDNSSEC).
//     This lets the caller fail fast with a clear "unsupported" error
//     before doing any work, rather than after a partial operation.
//  2. Type-assert the Provider to ZoneConfigurer to make the calls.
//
// A provider that sets a capability flag true MUST implement this
// interface; the flag is a contract, not just documentation.
type ZoneConfigurer interface {
	// GetZoneSettings returns the current value of the zone settings this
	// package manages (ssl, min_tls_version, tls_1_3, always_use_https,
	// automatic_https_rewrites), keyed by Cloudflare setting ID. Settings
	// this package does not manage are omitted even if the provider
	// exposes more.
	GetZoneSettings(ctx context.Context, zoneID string) (ZoneSettings, error)

	// SetZoneSettings updates only the settings present in the map,
	// leaving all others untouched.
	SetZoneSettings(ctx context.Context, zoneID string, settings ZoneSettings) error

	// GetBotManagement returns the zone's current bot management
	// configuration.
	GetBotManagement(ctx context.Context, zoneID string) (BotManagement, error)

	// SetBotManagement updates only the non-nil fields of bm, performing
	// a read-modify-write so fields the provider exposes but this
	// package does not model are preserved unchanged.
	SetBotManagement(ctx context.Context, zoneID string, bm BotManagement) error

	// GetSecurityTXT returns the zone's security.txt configuration.
	GetSecurityTXT(ctx context.Context, zoneID string) (SecurityTXT, error)

	// SetSecurityTXT replaces the zone's security.txt configuration.
	SetSecurityTXT(ctx context.Context, zoneID string, txt SecurityTXT) error

	// GetRedirectRules returns the redirect rules in the zone's dynamic
	// redirect ruleset whose Ref carries refPrefix. Rules owned by other
	// callers (any Ref not carrying refPrefix) are never returned.
	GetRedirectRules(ctx context.Context, zoneID, refPrefix string) ([]RedirectRule, error)

	// SetRedirectRules replaces the redirect rules owned by refPrefix
	// with rules, leaving every other rule in the ruleset - including
	// rules with no ref at all - untouched and in place.
	SetRedirectRules(ctx context.Context, zoneID, refPrefix string, rules []RedirectRule) error

	// GetDNSSEC returns the zone's current DNSSEC status and DS record.
	GetDNSSEC(ctx context.Context, zoneID string) (DNSSECStatus, error)

	// SetDNSSEC enables or disables DNSSEC for the zone and returns the
	// resulting status.
	SetDNSSEC(ctx context.Context, zoneID string, enabled bool) (DNSSECStatus, error)
}

// ZoneSettings maps a Cloudflare zone setting ID (e.g. "ssl",
// "min_tls_version") to its value. Values are the provider's native JSON
// types (string, bool, number) so callers should type-switch on read.
type ZoneSettings map[string]interface{}

// BotManagement is a partial view of a zone's bot management
// configuration. All fields are pointers: nil means "leave unchanged" on
// a Set call, and Get always populates every field it recognizes.
type BotManagement struct {
	FightMode         *bool
	CrawlerProtection *bool
	EnableJS          *bool
}

// SecurityTXT is a zone's security.txt (RFC 9116) configuration.
type SecurityTXT struct {
	Enabled            bool
	Contact            []string
	Expires            string
	Encryption         []string
	Acknowledgments    []string
	PreferredLanguages []string
	Policy             []string
	Hiring             []string
	CanonicalURI       []string
}

// RedirectRule is one entry in a zone's http_request_dynamic_redirect
// ruleset, restricted to the fields this package manages.
type RedirectRule struct {
	// Ref uniquely identifies the rule and marks ownership: callers pass
	// their own prefix to GetRedirectRules/SetRedirectRules, and only
	// rules whose Ref carries that prefix are read, replaced, or deleted.
	Ref                 string
	Description         string
	Expression          string
	TargetExpression    string
	StatusCode          int
	PreserveQueryString bool
	Enabled             bool
}

// DNSSECStatus reports a zone's DNSSEC state and signing material.
type DNSSECStatus struct {
	Status     string
	DS         string
	KeyTag     int
	Algorithm  string
	DigestType string
	Digest     string
	PublicKey  string
}
