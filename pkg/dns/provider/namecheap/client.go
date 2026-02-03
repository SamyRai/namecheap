package namecheap

import (
	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

// NamecheapClient defines the interface for Namecheap API operations
type NamecheapClient interface {
	DomainsGetList(args *namecheap.DomainsGetListArgs) (*namecheap.DomainsGetListCommandResponse, error)
	DomainsGetInfo(domainName string) (*namecheap.DomainsGetInfoCommandResponse, error)
	DomainsDNSGetHosts(domainName string) (*namecheap.DomainsDNSGetHostsCommandResponse, error)
	DomainsDNSSetHosts(args *namecheap.DomainsDNSSetHostsArgs) (*namecheap.DomainsDNSSetHostsCommandResponse, error)
}

// SDKClient wraps the official Namecheap SDK client
type SDKClient struct {
	client *namecheap.Client
}

// NewSDKClient creates a new SDKClient wrapper
func NewSDKClient(client *namecheap.Client) *SDKClient {
	return &SDKClient{client: client}
}

func (c *SDKClient) DomainsGetList(args *namecheap.DomainsGetListArgs) (*namecheap.DomainsGetListCommandResponse, error) {
	return c.client.Domains.GetList(args)
}

func (c *SDKClient) DomainsGetInfo(domainName string) (*namecheap.DomainsGetInfoCommandResponse, error) {
	return c.client.Domains.GetInfo(domainName)
}

func (c *SDKClient) DomainsDNSGetHosts(domainName string) (*namecheap.DomainsDNSGetHostsCommandResponse, error) {
	return c.client.DomainsDNS.GetHosts(domainName)
}

func (c *SDKClient) DomainsDNSSetHosts(args *namecheap.DomainsDNSSetHostsArgs) (*namecheap.DomainsDNSSetHostsCommandResponse, error) {
	return c.client.DomainsDNS.SetHosts(args)
}
