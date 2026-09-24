package cloudflare

import (
	"testing"

	"zonekit/pkg/config"

	"github.com/stretchr/testify/require"
)

func TestFromAccount_DefaultsToAccountScope(t *testing.T) {
	cfg, err := FromAccount(&config.AccountConfig{
		Provider:  config.ProviderCloudflare,
		APIToken:  testAPIToken,
		AccountID: "acct-1",
	})
	require.NoError(t, err)
	require.Equal(t, testAPIToken, cfg.APIToken)
	require.Equal(t, "acct-1", cfg.AccountID)
	require.Equal(t, TokenScopeAccount, cfg.TokenScope)
}

func TestFromAccount_UserScope(t *testing.T) {
	cfg, err := FromAccount(&config.AccountConfig{
		Provider:   config.ProviderCloudflare,
		APIToken:   testAPIToken,
		TokenScope: "user",
	})
	require.NoError(t, err)
	require.Equal(t, TokenScopeUser, cfg.TokenScope)
}

func TestFromAccount_NilAccount(t *testing.T) {
	_, err := FromAccount(nil)
	require.Error(t, err)
}

func TestNewFromAccount(t *testing.T) {
	p, err := NewFromAccount(&config.AccountConfig{
		Provider:  config.ProviderCloudflare,
		APIToken:  testAPIToken,
		AccountID: "acct-1",
	})
	require.NoError(t, err)
	require.NoError(t, p.Validate())
}
