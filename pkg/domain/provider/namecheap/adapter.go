package namecheap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"zonekit/pkg/client"
	"zonekit/pkg/domain/model"
	"zonekit/pkg/domain/provider"
	"zonekit/pkg/pointer"

	nc "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

// Adapter implements the Domain Provider interface for Namecheap
type Adapter struct {
	client *client.Client
	nc     *nc.Client
}

func init() {
	provider.Register("namecheap", New)
}

// New creates a new Namecheap domain provider
func New(c *client.Client) (provider.Provider, error) {
	return &Adapter{
		client: c,
		nc:     c.GetNamecheapClient(),
	}, nil
}

func (a *Adapter) Name() string {
	return "namecheap"
}

func (a *Adapter) ListDomains(ctx context.Context) ([]model.Domain, error) {
	resp, err := a.nc.Domains.GetList(&nc.DomainsGetListArgs{
		ListType: nc.String("ALL"),
		Page:     nc.Int(1),
		PageSize: nc.Int(100),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get domain list: %w", err)
	}

	var domains []model.Domain
	for _, d := range *resp.Domains {
		domains = append(domains, model.Domain{
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
		})
	}
	return domains, nil
}

func (a *Adapter) GetDomainInfo(ctx context.Context, domainName string) (*model.Domain, error) {
	resp, err := a.nc.Domains.GetInfo(domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain info: %w", err)
	}

	return &model.Domain{
		Name:       pointer.String(resp.DomainDNSGetListResult.DomainName),
		IsExpired:  false,
		IsLocked:   false,
		AutoRenew:  false,
		WhoisGuard: "",
		IsPremium:  pointer.Bool(resp.DomainDNSGetListResult.IsPremium),
		IsOurDNS:   pointer.Bool(resp.DomainDNSGetListResult.DnsDetails.IsUsingOurDNS),
	}, nil
}

func (a *Adapter) CheckAvailability(ctx context.Context, domainName string) (bool, error) {
	// Bypass SDK to call namecheap.domains.check directly via HTTP
	config := a.client.GetConfig()
	apiURL := "https://api.namecheap.com/xml.response"
	if config.UseSandbox {
		apiURL = "https://api.sandbox.namecheap.com/xml.response"
	}

	params := url.Values{}
	params.Set("ApiUser", config.APIUser)
	params.Set("ApiKey", config.APIKey)
	params.Set("UserName", config.Username)
	params.Set("ClientIp", config.ClientIP)
	params.Set("Command", "namecheap.domains.check")
	params.Set("DomainList", domainName)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	bodyStr := string(body)
	if strings.Contains(bodyStr, `Available="true"`) {
		return true, nil
	} else if strings.Contains(bodyStr, `Available="false"`) {
		return false, nil
	}
	
	if strings.Contains(bodyStr, "<Errors>") && !strings.Contains(bodyStr, "<Errors />") {
		return false, fmt.Errorf("namecheap API error: %s", bodyStr)
	}

	return false, fmt.Errorf("could not determine availability from response")
}

func (a *Adapter) RegisterDomain(ctx context.Context, req model.RegistrationRequest) error {
	config := a.client.GetConfig()
	apiURL := "https://api.namecheap.com/xml.response"
	if config.UseSandbox {
		apiURL = "https://api.sandbox.namecheap.com/xml.response"
	}

	params := url.Values{}
	params.Set("ApiUser", config.APIUser)
	params.Set("ApiKey", config.APIKey)
	params.Set("UserName", config.Username)
	params.Set("ClientIp", config.ClientIP)
	params.Set("Command", "namecheap.domains.create")
	params.Set("DomainName", req.DomainName)
	params.Set("Years", fmt.Sprintf("%d", req.Years))

	// Map contact info helper
	setContact := func(prefix string, c model.ContactInfo) {
		params.Set(prefix+"FirstName", c.FirstName)
		params.Set(prefix+"LastName", c.LastName)
		params.Set(prefix+"Address1", c.Address)
		params.Set(prefix+"City", c.City)
		params.Set(prefix+"StateProvince", c.StateProvince)
		params.Set(prefix+"PostalCode", c.PostalCode)
		params.Set(prefix+"Country", c.Country)
		params.Set(prefix+"Phone", c.Phone)
		params.Set(prefix+"EmailAddress", c.Email)
	}

	setContact("Registrant", req.Registrant)
	setContact("Tech", req.Tech)
	setContact("Admin", req.Admin)
	setContact("AuxBilling", req.AuxBilling)

	reqHTTP, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	reqHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyStr := string(body)
	if strings.Contains(bodyStr, "<Errors>") && !strings.Contains(bodyStr, "<Errors />") {
		return fmt.Errorf("namecheap API error during registration: %s", bodyStr)
	}

	if !strings.Contains(bodyStr, `Registered="true"`) {
		return fmt.Errorf("domain registration failed, unexpected response: %s", bodyStr)
	}

	return nil
}

func (a *Adapter) RenewDomain(ctx context.Context, domainName string, years int) error {
	config := a.client.GetConfig()
	apiURL := "https://api.namecheap.com/xml.response"
	if config.UseSandbox {
		apiURL = "https://api.sandbox.namecheap.com/xml.response"
	}

	params := url.Values{}
	params.Set("ApiUser", config.APIUser)
	params.Set("ApiKey", config.APIKey)
	params.Set("UserName", config.Username)
	params.Set("ClientIp", config.ClientIP)
	params.Set("Command", "namecheap.domains.renew")
	params.Set("DomainName", domainName)
	params.Set("Years", fmt.Sprintf("%d", years))

	reqHTTP, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	reqHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyStr := string(body)
	if strings.Contains(bodyStr, "<Errors>") && !strings.Contains(bodyStr, "<Errors />") {
		return fmt.Errorf("namecheap API error during renewal: %s", bodyStr)
	}

	if !strings.Contains(bodyStr, `Renew="true"`) {
		return fmt.Errorf("domain renewal failed, unexpected response: %s", bodyStr)
	}

	return nil
}

func (a *Adapter) GetNameservers(ctx context.Context, domainName string) ([]string, error) {
	resp, err := a.nc.DomainsDNS.GetList(domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get nameservers: %w", err)
	}
	var ns []string
	if resp.DomainDNSGetListResult != nil && resp.DomainDNSGetListResult.Nameservers != nil {
		ns = append(ns, *resp.DomainDNSGetListResult.Nameservers...)
	}
	return ns, nil
}

func (a *Adapter) SetNameservers(ctx context.Context, domainName string, nameservers []string) error {
	_, err := a.nc.DomainsDNS.SetCustom(domainName, nameservers)
	return err
}

func (a *Adapter) SetDefaultNameservers(ctx context.Context, domainName string) error {
	_, err := a.nc.DomainsDNS.SetDefault(domainName)
	return err
}

func (a *Adapter) EnableDNSSEC(ctx context.Context, domainName string) error {
	return a.doDNSSECAction(ctx, domainName, "namecheap.domains.dns.setDnssec", "add")
}

func (a *Adapter) DisableDNSSEC(ctx context.Context, domainName string) error {
	return a.doDNSSECAction(ctx, domainName, "namecheap.domains.dns.setDnssec", "del")
}

func (a *Adapter) GetDNSSECStatus(ctx context.Context, domainName string) (bool, error) {
	// Status checking requires parsing XML which is tedious without SDK.
	// Returning unsupported for now.
	return false, fmt.Errorf("namecheap get dnssec status not fully implemented")
}

func (a *Adapter) doDNSSECAction(ctx context.Context, domainName, command, action string) error {
	config := a.client.GetConfig()
	apiURL := "https://api.namecheap.com/xml.response"
	if config.UseSandbox {
		apiURL = "https://api.sandbox.namecheap.com/xml.response"
	}

	params := url.Values{}
	params.Set("ApiUser", config.APIUser)
	params.Set("ApiKey", config.APIKey)
	params.Set("UserName", config.Username)
	params.Set("ClientIp", config.ClientIP)
	params.Set("Command", command)
	params.Set("DomainName", domainName)
	if action != "" {
		params.Set("Action", action)
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	reqHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyStr := string(body)
	if strings.Contains(bodyStr, "<Errors>") && !strings.Contains(bodyStr, "<Errors />") {
		return fmt.Errorf("namecheap API error during dnssec operation: %s", bodyStr)
	}

	return nil
}

func getDateTime(dt *nc.DateTime) string {
	if dt == nil {
		return ""
	}
	return dt.String()
}
