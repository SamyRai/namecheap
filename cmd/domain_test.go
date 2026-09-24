package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"go.glpx.pro/zonekit/internal/cmdutil"
	"go.glpx.pro/zonekit/pkg/domain"
)

// O1: domain list/info must emit stable snake_case fields. domain.Service
// talks to the real Namecheap SDK directly (no injectable factory exists,
// unlike dns.Service), so these test the DTO conversion and its wire shape
// directly rather than driving the full CLI, to avoid any network access.
func TestDomainDTO_SnakeCaseFields(t *testing.T) {
	d := domain.Domain{
		Name:       "example.com",
		User:       "alice",
		Created:    "2024-01-01",
		Expires:    "2025-01-01",
		IsExpired:  false,
		IsLocked:   true,
		AutoRenew:  true,
		WhoisGuard: "ENABLED",
		IsPremium:  false,
		IsOurDNS:   true,
	}

	dto := newDomainDTO(d)

	var buf bytes.Buffer
	require.NoError(t, cmdutil.WriteResult(&buf, cmdutil.OutputJSON, dto))

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.Equal(t, "example.com", decoded["name"])
	require.Equal(t, "alice", decoded["user"])
	require.Equal(t, true, decoded["is_locked"])
	require.Equal(t, true, decoded["auto_renew"])
	require.Equal(t, true, decoded["is_our_dns"])
	require.Equal(t, false, decoded["is_premium"])
}

func TestDomainDTOs_PreservesOrderAndCount(t *testing.T) {
	domains := []domain.Domain{
		{Name: "a.com"},
		{Name: "b.com"},
	}
	dtos := newDomainDTOs(domains)
	require.Len(t, dtos, 2)
	require.Equal(t, "a.com", dtos[0].Name)
	require.Equal(t, "b.com", dtos[1].Name)
}
