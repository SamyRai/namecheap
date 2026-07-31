package all

import (
	"slices"
	"testing"

	"zonekit/pkg/domain/provider"
)

// Every provider that ships must be registered. Before this package existed,
// pkg/domain/provider/namecheap was imported by nothing, so its init() never
// ran and the whole `domain` command tree failed at runtime with
// "provider 'namecheap' not found" -- a failure no compiler check catches.
func TestAllProvidersAreRegistered(t *testing.T) {
	registered := provider.List()
	for _, name := range []string{"namecheap", "cloudflare", "digitalocean", "godaddy"} {
		if !slices.Contains(registered, name) {
			t.Errorf("provider %q is not registered (registered: %v)", name, registered)
		}
	}
}
