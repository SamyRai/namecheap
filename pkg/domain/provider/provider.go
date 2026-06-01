package provider

import (
	"context"

	"zonekit/pkg/domain/model"
)

// Provider defines the interface that all Domain providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "namecheap", "cloudflare")
	Name() string

	// ListDomains retrieves all domains for the authenticated user
	ListDomains(ctx context.Context) ([]model.Domain, error)

	// GetDomainInfo retrieves detailed information about a specific domain
	GetDomainInfo(ctx context.Context, domainName string) (*model.Domain, error)

	// CheckAvailability checks if a domain is available for registration
	CheckAvailability(ctx context.Context, domainName string) (bool, error)

	// RegisterDomain registers a new domain
	RegisterDomain(ctx context.Context, req model.RegistrationRequest) error

	// RenewDomain renews an existing domain
	RenewDomain(ctx context.Context, domainName string, years int) error

	// GetNameservers retrieves the nameservers for a domain
	GetNameservers(ctx context.Context, domainName string) ([]string, error)

	// SetNameservers sets custom nameservers for a domain
	SetNameservers(ctx context.Context, domainName string, nameservers []string) error

	// SetDefaultNameservers sets the domain to use the provider's default DNS servers
	SetDefaultNameservers(ctx context.Context, domainName string) error

	// DNSSEC operations
	EnableDNSSEC(ctx context.Context, domainName string) error
	DisableDNSSEC(ctx context.Context, domainName string) error
	GetDNSSECStatus(ctx context.Context, domainName string) (bool, error)
}
