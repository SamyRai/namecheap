// Package all links every domain-registrar provider into the binary.
//
// Providers register themselves from init(), so a provider that nothing
// imports is never registered and its name resolves to
// "provider '<name>' not found" at runtime -- with no build error to catch it.
// Importing this package once, from the command layer, keeps that wiring in a
// single obvious place instead of scattering blank imports across commands.
package all

import (
	// Registered for their side effects.
	_ "zonekit/pkg/domain/provider/cloudflare"
	_ "zonekit/pkg/domain/provider/digitalocean"
	_ "zonekit/pkg/domain/provider/godaddy"
	_ "zonekit/pkg/domain/provider/namecheap"
)
