package cloudflare

import (
	"testing"

	dnsprovider "zonekit/pkg/dns/provider"

	"github.com/stretchr/testify/require"
)

func TestRegister_ResolvesUnderCloudflareName(t *testing.T) {
	dnsprovider.Clear()
	t.Cleanup(dnsprovider.Clear)

	_, err := Register(Config{APIToken: testAPIToken, TokenScope: TokenScopeUser})
	require.NoError(t, err)

	got, err := dnsprovider.Get(providerName)
	require.NoError(t, err)
	require.Equal(t, providerName, got.Name())
}

func TestRegister_DuplicateFails(t *testing.T) {
	dnsprovider.Clear()
	t.Cleanup(dnsprovider.Clear)

	_, err := Register(Config{APIToken: testAPIToken, TokenScope: TokenScopeUser})
	require.NoError(t, err)

	_, err = Register(Config{APIToken: "tok2", TokenScope: TokenScopeUser})
	require.Error(t, err)
}
