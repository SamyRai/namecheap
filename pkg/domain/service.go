package domain

import (
	"context"
	"fmt"

	"zonekit/pkg/client"
	"zonekit/pkg/domain/model"
	"zonekit/pkg/domain/provider"
)

// Service provides domain management operations
type Service struct {
	provider provider.Provider
}

// NewService creates a new domain service
func NewService(client *client.Client) (*Service, error) {
	providerName := client.GetConfig().GetProvider()
	p, err := provider.Get(providerName, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain provider: %w", err)
	}

	return &Service{
		provider: p,
	}, nil
}

// ListDomains retrieves all domains for the authenticated user
func (s *Service) ListDomains(ctx context.Context) ([]model.Domain, error) {
	return s.provider.ListDomains(ctx)
}

// GetDomainInfo retrieves detailed information about a specific domain
func (s *Service) GetDomainInfo(ctx context.Context, domainName string) (*model.Domain, error) {
	return s.provider.GetDomainInfo(ctx, domainName)
}

// CheckAvailability checks if a domain is available for registration
func (s *Service) CheckAvailability(ctx context.Context, domainName string) (bool, error) {
	return s.provider.CheckAvailability(ctx, domainName)
}

// RegisterDomain registers a new domain
func (s *Service) RegisterDomain(ctx context.Context, req model.RegistrationRequest) error {
	return s.provider.RegisterDomain(ctx, req)
}

// RenewDomain renews an existing domain
func (s *Service) RenewDomain(ctx context.Context, domainName string, years int) error {
	return s.provider.RenewDomain(ctx, domainName, years)
}

// GetNameservers retrieves the nameservers for a domain
func (s *Service) GetNameservers(ctx context.Context, domainName string) ([]string, error) {
	return s.provider.GetNameservers(ctx, domainName)
}

// SetNameservers sets custom nameservers for a domain
func (s *Service) SetNameservers(ctx context.Context, domainName string, nameservers []string) error {
	return s.provider.SetNameservers(ctx, domainName, nameservers)
}

// SetDefaultNameservers sets the domain to use the provider's default DNS servers
func (s *Service) SetDefaultNameservers(ctx context.Context, domainName string) error {
	return s.provider.SetDefaultNameservers(ctx, domainName)
}

// EnableDNSSEC enables DNSSEC for a domain
func (s *Service) EnableDNSSEC(ctx context.Context, domainName string) error {
	return s.provider.EnableDNSSEC(ctx, domainName)
}

// DisableDNSSEC disables DNSSEC for a domain
func (s *Service) DisableDNSSEC(ctx context.Context, domainName string) error {
	return s.provider.DisableDNSSEC(ctx, domainName)
}

// GetDNSSECStatus returns whether DNSSEC is enabled
func (s *Service) GetDNSSECStatus(ctx context.Context, domainName string) (bool, error) {
	return s.provider.GetDNSSECStatus(ctx, domainName)
}

// Domain is deprecated, use model.Domain instead
type Domain = model.Domain
