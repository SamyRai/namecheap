package domain

import (
	"fmt"
	"strings"

	"zonekit/pkg/client"
	"zonekit/pkg/pointer"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

// Service provides domain management operations
type Service struct {
	client *client.Client
}

// NewService creates a new domain service
func NewService(client *client.Client) *Service {
	return &Service{
		client: client,
	}
}

// Domain represents a domain with its details
type Domain struct {
	Name       string
	User       string
	Created    string
	Expires    string
	IsExpired  bool
	IsLocked   bool
	AutoRenew  bool
	WhoisGuard string
	IsPremium  bool
	IsOurDNS   bool
}

// ListDomains retrieves all domains for the authenticated user
func (s *Service) ListDomains() ([]Domain, error) {
	nc := s.client.GetNamecheapClient()

	resp, err := nc.Domains.GetList(&namecheap.DomainsGetListArgs{
		ListType: namecheap.String("ALL"),
		Page:     namecheap.Int(1),
		PageSize: namecheap.Int(100),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get domain list: %w", err)
	}

	domains := make([]Domain, 0, len(*resp.Domains))
	for _, d := range *resp.Domains {
		domain := Domain{
			Name:       pointer.String(d.Name),
			User:       pointer.String(d.User),
			Created:    getDateTime(d.Created),
			Expires:    getDateTime(d.Expires),
			IsExpired:  pointer.Bool(d.IsExpired),
			IsLocked:   pointer.Bool(d.IsLocked),
			AutoRenew:  pointer.Bool(d.AutoRenew),
			WhoisGuard: pointer.String(d.WhoisGuard),
			IsPremium:  pointer.Bool(d.IsPremium),
			IsOurDNS:   pointer.Bool(d.IsOurDNS),
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// GetDomainInfo retrieves detailed information about a specific domain
func (s *Service) GetDomainInfo(domainName string) (*Domain, error) {
	nc := s.client.GetNamecheapClient()

	resp, err := nc.Domains.GetInfo(domainName)

	if err != nil {
		return nil, fmt.Errorf("failed to get domain info for %s: %w", domainName, err)
	}

	domain := &Domain{
		Name:       pointer.String(resp.DomainDNSGetListResult.DomainName),
		User:       "", // Not available in this response
		Created:    "", // Not available in this response
		Expires:    "", // Not available in this response
		IsExpired:  false,
		IsLocked:   false,
		AutoRenew:  false,
		WhoisGuard: "",
		IsPremium:  pointer.Bool(resp.DomainDNSGetListResult.IsPremium),
		IsOurDNS:   pointer.Bool(resp.DomainDNSGetListResult.DnsDetails.IsUsingOurDNS),
	}

	return domain, nil
}

// CheckAvailability checks if a domain is available for registration
func (s *Service) CheckAvailability(domainName string) (bool, error) {
	// Note: The current Namecheap SDK (v2.4.1) doesn't implement domain availability checking
	// This would require implementing a direct API call to the domains.check endpoint
	// For now, return an informative error
	return false, fmt.Errorf("domain availability check not supported by current SDK version - please check manually at namecheap.com")
}

// RegisterDomain registers a new domain (placeholder - needs contact info)
func (s *Service) RegisterDomain(domainName string) error {
	// TODO: Implement domain registration
	// This requires contact information and payment details
	return fmt.Errorf("domain registration not yet implemented - requires contact info and payment setup")
}

// RenewDomain renews an existing domain
func (s *Service) RenewDomain(domainName string, years int) error {
	// TODO: Implement domain renewal
	// The SDK doesn't seem to have a direct renew method in the current version
	return fmt.Errorf("domain renewal not yet implemented - TODO: add domains.renew API")
}

// GetNameservers retrieves the nameservers for a domain
func (s *Service) GetNameservers(domainName string) ([]string, error) {
	nc := s.client.GetNamecheapClient()

	resp, err := nc.DomainsDNS.GetList(domainName)

	if err != nil {
		return nil, fmt.Errorf("failed to get nameservers for %s: %w", domainName, err)
	}

	nameservers := make([]string, 0, len(*resp.DomainDNSGetListResult.Nameservers))
	for _, ns := range *resp.DomainDNSGetListResult.Nameservers {
		nameservers = append(nameservers, ns)
	}

	return nameservers, nil
}

// SetNameservers sets custom nameservers for a domain
func (s *Service) SetNameservers(domainName string, nameservers []string) error {
	nc := s.client.GetNamecheapClient()

	_, err := nc.DomainsDNS.SetCustom(domainName, nameservers)

	if err != nil {
		return fmt.Errorf("failed to set nameservers for %s: %w", domainName, err)
	}

	return nil
}

// SetToNamecheapDNS sets the domain to use Namecheap's DNS servers
func (s *Service) SetToNamecheapDNS(domainName string) error {
	nc := s.client.GetNamecheapClient()

	_, err := nc.DomainsDNS.SetDefault(domainName)

	if err != nil {
		return fmt.Errorf("failed to set domain %s to use Namecheap DNS: %w", domainName, err)
	}

	return nil
}

func getDateTime(dt *namecheap.DateTime) string {
	if dt == nil {
		return ""
	}
	return dt.String()
}

func getDomain(fullDomain string) string {
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 2 {
		return fullDomain
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

func getTLD(fullDomain string) string {
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}
