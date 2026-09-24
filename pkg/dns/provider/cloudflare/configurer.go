package cloudflare

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	dnsprovider "go.glpx.pro/zonekit/pkg/dns/provider"
)

// Compile-time assertion that Provider satisfies the optional
// ZoneConfigurer capability advertised by its Capabilities().
var _ dnsprovider.ZoneConfigurer = (*Provider)(nil)

// Zone setting IDs GetZoneSettings/SetZoneSettings read and write.
const (
	SettingSSL                    = "ssl"
	SettingMinTLSVersion          = "min_tls_version"
	SettingTLS13                  = "tls_1_3"
	SettingAlwaysUseHTTPS         = "always_use_https"
	SettingAutomaticHTTPSRewrites = "automatic_https_rewrites"
)

// managedZoneSettings lists the settings above. Cloudflare's GET
// /zones/{id}/settings returns dozens of settings; this package only
// manages this security-relevant subset, and leaves everything else
// alone.
var managedZoneSettings = []string{
	SettingSSL,
	SettingMinTLSVersion,
	SettingTLS13,
	SettingAlwaysUseHTTPS,
	SettingAutomaticHTTPSRewrites,
}

type zoneSettingWire struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value"`
}

// GetZoneSettings implements dnsprovider.ZoneConfigurer.
func (p *Provider) GetZoneSettings(ctx context.Context, zoneID string) (dnsprovider.ZoneSettings, error) {
	var all []zoneSettingWire
	if err := p.client.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/settings", &all); err != nil {
		return nil, fmt.Errorf("cloudflare: get zone settings: %w", err)
	}

	managed := make(map[string]bool, len(managedZoneSettings))
	for _, id := range managedZoneSettings {
		managed[id] = true
	}

	out := dnsprovider.ZoneSettings{}
	for _, s := range all {
		if managed[s.ID] {
			out[s.ID] = s.Value
		}
	}
	return out, nil
}

// SetZoneSettings implements dnsprovider.ZoneConfigurer. It PATCHes only
// the settings present in the map via the bulk settings endpoint's
// `items` array, leaving every other setting - managed or not -
// untouched.
func (p *Provider) SetZoneSettings(ctx context.Context, zoneID string, settings dnsprovider.ZoneSettings) error {
	if len(settings) == 0 {
		return nil
	}

	items := make([]zoneSettingWire, 0, len(settings))
	for id, value := range settings {
		items = append(items, zoneSettingWire{ID: id, Value: value})
	}

	body := map[string]interface{}{"items": items}
	if err := p.client.patch(ctx, "/zones/"+url.PathEscape(zoneID)+"/settings", body, nil); err != nil {
		return fmt.Errorf("cloudflare: set zone settings: %w", err)
	}
	return nil
}

// GetBotManagement implements dnsprovider.ZoneConfigurer.
func (p *Provider) GetBotManagement(ctx context.Context, zoneID string) (dnsprovider.BotManagement, error) {
	raw, err := p.getBotManagementRaw(ctx, zoneID)
	if err != nil {
		return dnsprovider.BotManagement{}, err
	}
	return botManagementFromRaw(raw), nil
}

// SetBotManagement implements dnsprovider.ZoneConfigurer as a
// read-modify-write: it fetches the current configuration, overlays only
// the non-nil fields of bm onto it, and PUTs the full object back so
// fields this package does not model (e.g. sbfm_* scores) survive
// unchanged.
func (p *Provider) SetBotManagement(ctx context.Context, zoneID string, bm dnsprovider.BotManagement) error {
	raw, err := p.getBotManagementRaw(ctx, zoneID)
	if err != nil {
		return err
	}

	if bm.FightMode != nil {
		raw["fight_mode"] = *bm.FightMode
	}
	if bm.CrawlerProtection != nil {
		raw["crawler_protection"] = *bm.CrawlerProtection
	}
	if bm.EnableJS != nil {
		raw["enable_js"] = *bm.EnableJS
	}

	if err := p.client.put(ctx, "/zones/"+url.PathEscape(zoneID)+"/bot_management", raw, nil); err != nil {
		return fmt.Errorf("cloudflare: set bot management: %w", err)
	}
	return nil
}

func (p *Provider) getBotManagementRaw(ctx context.Context, zoneID string) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := p.client.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/bot_management", &raw); err != nil {
		return nil, fmt.Errorf("cloudflare: get bot management: %w", err)
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}
	return raw, nil
}

func botManagementFromRaw(raw map[string]interface{}) dnsprovider.BotManagement {
	var bm dnsprovider.BotManagement
	if v, ok := raw["fight_mode"].(bool); ok {
		bm.FightMode = &v
	}
	if v, ok := raw["crawler_protection"].(bool); ok {
		bm.CrawlerProtection = &v
	}
	if v, ok := raw["enable_js"].(bool); ok {
		bm.EnableJS = &v
	}
	return bm
}

// securityTXTWire is the Cloudflare security-center securitytxt wire
// shape (snake_case field names, per the API).
type securityTXTWire struct {
	Enabled            bool     `json:"enabled"`
	Contact            []string `json:"contact,omitempty"`
	Expires            string   `json:"expires,omitempty"`
	Encryption         []string `json:"encryption,omitempty"`
	Acknowledgments    []string `json:"acknowledgments,omitempty"`
	PreferredLanguages []string `json:"preferred_languages,omitempty"`
	Policy             []string `json:"policy,omitempty"`
	Hiring             []string `json:"hiring,omitempty"`
	Canonical          []string `json:"canonical,omitempty"`
}

// GetSecurityTXT implements dnsprovider.ZoneConfigurer.
func (p *Provider) GetSecurityTXT(ctx context.Context, zoneID string) (dnsprovider.SecurityTXT, error) {
	var w securityTXTWire
	if err := p.client.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/security-center/securitytxt", &w); err != nil {
		return dnsprovider.SecurityTXT{}, fmt.Errorf("cloudflare: get security.txt: %w", err)
	}
	return dnsprovider.SecurityTXT{
		Enabled:            w.Enabled,
		Contact:            w.Contact,
		Expires:            w.Expires,
		Encryption:         w.Encryption,
		Acknowledgments:    w.Acknowledgments,
		PreferredLanguages: w.PreferredLanguages,
		Policy:             w.Policy,
		Hiring:             w.Hiring,
		CanonicalURI:       w.Canonical,
	}, nil
}

// SetSecurityTXT implements dnsprovider.ZoneConfigurer.
func (p *Provider) SetSecurityTXT(ctx context.Context, zoneID string, txt dnsprovider.SecurityTXT) error {
	w := securityTXTWire{
		Enabled:            txt.Enabled,
		Contact:            txt.Contact,
		Expires:            txt.Expires,
		Encryption:         txt.Encryption,
		Acknowledgments:    txt.Acknowledgments,
		PreferredLanguages: txt.PreferredLanguages,
		Policy:             txt.Policy,
		Hiring:             txt.Hiring,
		Canonical:          txt.CanonicalURI,
	}
	if err := p.client.put(ctx, "/zones/"+url.PathEscape(zoneID)+"/security-center/securitytxt", w, nil); err != nil {
		return fmt.Errorf("cloudflare: set security.txt: %w", err)
	}
	return nil
}

// redirectRulesetPath is the phase entrypoint for a zone's dynamic
// redirect ruleset.
func redirectRulesetPath(zoneID string) string {
	return "/zones/" + url.PathEscape(zoneID) + "/rulesets/phases/http_request_dynamic_redirect/entrypoint"
}

type rulesetWire struct {
	ID    string                   `json:"id,omitempty"`
	Rules []map[string]interface{} `json:"rules"`
}

// GetRedirectRules implements dnsprovider.ZoneConfigurer.
func (p *Provider) GetRedirectRules(ctx context.Context, zoneID, refPrefix string) ([]dnsprovider.RedirectRule, error) {
	rules, err := p.getRedirectRuleset(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	var out []dnsprovider.RedirectRule
	for _, raw := range rules {
		ref, _ := raw[redirectFieldRef].(string)
		if refPrefix == "" || !strings.HasPrefix(ref, refPrefix) {
			continue
		}
		out = append(out, redirectRuleFromWire(raw))
	}
	return out, nil
}

// SetRedirectRules implements dnsprovider.ZoneConfigurer. It fetches the
// current ruleset, keeps every rule that is NOT ours (any rule whose ref
// does not carry refPrefix, including rules with no ref at all) exactly
// as returned by the API, and appends our replacement set built from
// rules. This is the only way to guarantee a caller never drops another
// system's redirect rules: we never round-trip foreign rules through our
// own (lossy) RedirectRule type, only pass their raw JSON through
// untouched.
func (p *Provider) SetRedirectRules(ctx context.Context, zoneID, refPrefix string, rules []dnsprovider.RedirectRule) error {
	if refPrefix == "" {
		return fmt.Errorf("cloudflare: refPrefix is required so foreign redirect rules are never touched")
	}

	existing, err := p.getRedirectRuleset(ctx, zoneID)
	if err != nil {
		return err
	}

	merged := make([]map[string]interface{}, 0, len(existing)+len(rules))
	for _, raw := range existing {
		ref, _ := raw[redirectFieldRef].(string)
		if strings.HasPrefix(ref, refPrefix) {
			continue // ours - dropped here, replaced by the loop below
		}
		merged = append(merged, raw)
	}
	for _, r := range rules {
		merged = append(merged, redirectRuleToWire(r))
	}

	body := rulesetWire{Rules: merged}
	if err := p.client.put(ctx, redirectRulesetPath(zoneID), body, nil); err != nil {
		return fmt.Errorf("cloudflare: set redirect rules: %w", err)
	}
	return nil
}

func (p *Provider) getRedirectRuleset(ctx context.Context, zoneID string) ([]map[string]interface{}, error) {
	var rs rulesetWire
	if err := p.client.get(ctx, redirectRulesetPath(zoneID), &rs); err != nil {
		return nil, fmt.Errorf("cloudflare: get redirect ruleset: %w", err)
	}
	return rs.Rules, nil
}

// Redirect rule wire field names, shared between redirectRuleToWire and
// redirectRuleFromWire so the two stay in sync.
const (
	redirectFieldRef         = "ref"
	redirectFieldDescription = "description"
	redirectFieldExpression  = "expression"
	redirectFieldEnabled     = "enabled"
	redirectFieldAction      = "action"
	redirectActionRedirect   = "redirect"
)

func redirectRuleToWire(r dnsprovider.RedirectRule) map[string]interface{} {
	return map[string]interface{}{
		redirectFieldRef:         r.Ref,
		redirectFieldDescription: r.Description,
		redirectFieldExpression:  r.Expression,
		redirectFieldEnabled:     r.Enabled,
		redirectFieldAction:      redirectActionRedirect,
		"action_parameters": map[string]interface{}{
			"from_value": map[string]interface{}{
				"status_code":           r.StatusCode,
				"preserve_query_string": r.PreserveQueryString,
				"target_url": map[string]interface{}{
					redirectFieldExpression: r.TargetExpression,
				},
			},
		},
	}
}

func redirectRuleFromWire(raw map[string]interface{}) dnsprovider.RedirectRule {
	r := dnsprovider.RedirectRule{}
	r.Ref, _ = raw[redirectFieldRef].(string)
	r.Description, _ = raw[redirectFieldDescription].(string)
	r.Expression, _ = raw[redirectFieldExpression].(string)
	r.Enabled, _ = raw[redirectFieldEnabled].(bool)

	params, _ := raw["action_parameters"].(map[string]interface{})
	fromValue, _ := params["from_value"].(map[string]interface{})
	if fromValue != nil {
		if sc, ok := fromValue["status_code"].(float64); ok {
			r.StatusCode = int(sc)
		}
		r.PreserveQueryString, _ = fromValue["preserve_query_string"].(bool)
		if target, ok := fromValue["target_url"].(map[string]interface{}); ok {
			r.TargetExpression, _ = target[redirectFieldExpression].(string)
		}
	}
	return r
}

type dnssecWire struct {
	Status     string `json:"status"`
	DS         string `json:"ds"`
	KeyTag     int    `json:"key_tag"`
	Algorithm  string `json:"algorithm"`
	DigestType string `json:"digest_type"`
	Digest     string `json:"digest"`
	PublicKey  string `json:"public_key"`
}

// GetDNSSEC implements dnsprovider.ZoneConfigurer.
func (p *Provider) GetDNSSEC(ctx context.Context, zoneID string) (dnsprovider.DNSSECStatus, error) {
	var w dnssecWire
	if err := p.client.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/dnssec", &w); err != nil {
		return dnsprovider.DNSSECStatus{}, fmt.Errorf("cloudflare: get dnssec: %w", err)
	}
	return dnssecFromWire(w), nil
}

// SetDNSSEC implements dnsprovider.ZoneConfigurer.
func (p *Provider) SetDNSSEC(ctx context.Context, zoneID string, enabled bool) (dnsprovider.DNSSECStatus, error) {
	status := "disabled"
	if enabled {
		status = "active"
	}

	var w dnssecWire
	body := map[string]string{"status": status}
	if err := p.client.patch(ctx, "/zones/"+url.PathEscape(zoneID)+"/dnssec", body, &w); err != nil {
		return dnsprovider.DNSSECStatus{}, fmt.Errorf("cloudflare: set dnssec: %w", err)
	}
	return dnssecFromWire(w), nil
}

func dnssecFromWire(w dnssecWire) dnsprovider.DNSSECStatus {
	return dnsprovider.DNSSECStatus{
		Status:     w.Status,
		DS:         w.DS,
		KeyTag:     w.KeyTag,
		Algorithm:  w.Algorithm,
		DigestType: w.DigestType,
		Digest:     w.Digest,
		PublicKey:  w.PublicKey,
	}
}
