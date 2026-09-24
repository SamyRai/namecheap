// Package cloudflare implements a typed zonekit DNS provider for
// Cloudflare, using a hand-rolled net/http client (no cloudflare-go
// dependency). It covers DNS record CRUD plus the optional
// provider.ZoneConfigurer capability (zone settings, bot management,
// security.txt, redirect rules, DNSSEC) that Cloudflare's Free plan
// exposes.
package cloudflare

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.glpx.pro/zonekit/pkg/config"
)

// TokenScope selects which Cloudflare endpoint a token verifies against.
type TokenScope string

const (
	// TokenScopeAccount verifies via /accounts/{account_id}/tokens/verify.
	// This is the default: account-owned tokens are what we use for 15
	// of our 17 zones.
	TokenScopeAccount TokenScope = "account"
	// TokenScopeUser verifies via /user/tokens/verify.
	TokenScopeUser TokenScope = "user"
)

// DefaultBaseURL is Cloudflare's production API v4 base URL.
const DefaultBaseURL = "https://api.cloudflare.com/client/v4"

// DefaultTimeout bounds each HTTP request when Config.Timeout is unset.
const DefaultTimeout = 30 * time.Second

// Config configures a Cloudflare DNS provider instance.
type Config struct {
	// APIToken is a Cloudflare API token (Bearer auth). Required.
	APIToken string
	// AccountID is the Cloudflare account ID. Required when TokenScope
	// is TokenScopeAccount (the default).
	AccountID string
	// TokenScope selects the token-verification endpoint. Defaults to
	// TokenScopeAccount.
	TokenScope TokenScope
	// BaseURL overrides the Cloudflare API base URL. Defaults to
	// DefaultBaseURL; tests point this at an httptest server.
	BaseURL string
	// Timeout bounds each HTTP request. Defaults to DefaultTimeout.
	Timeout time.Duration
	// HTTPClient overrides the underlying http.Client. When set, Timeout
	// is ignored - the caller owns the client's timeout policy.
	HTTPClient *http.Client
}

// FromAccount builds a Cloudflare Config from a zonekit account
// configuration. The account's Provider must be "cloudflare" (or empty is
// rejected by the caller before this is reached - FromAccount does not
// re-check it, since a caller may want to build a Config for an account
// whose Provider field was set by convention elsewhere).
func FromAccount(account *config.AccountConfig) (Config, error) {
	if account == nil {
		return Config{}, fmt.Errorf("cloudflare: account configuration is nil")
	}

	scope := TokenScopeAccount
	if account.TokenScope == string(TokenScopeUser) {
		scope = TokenScopeUser
	}

	return Config{
		APIToken:   account.APIToken,
		AccountID:  account.AccountID,
		TokenScope: scope,
	}, nil
}

func (c Config) baseURL() string {
	if c.BaseURL == "" {
		return DefaultBaseURL
	}
	return strings.TrimRight(c.BaseURL, "/")
}

func (c Config) tokenScope() TokenScope {
	if c.TokenScope == "" {
		return TokenScopeAccount
	}
	return c.TokenScope
}

func (c Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &http.Client{Timeout: timeout}
}
