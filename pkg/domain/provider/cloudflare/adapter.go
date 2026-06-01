package cloudflare

import (
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"zonekit/pkg/client"
	"zonekit/pkg/domain/model"
	"zonekit/pkg/domain/provider"
)

type Adapter struct {
	client *client.Client
}

func init() {
	provider.Register("cloudflare", New)
}

func New(c *client.Client) (provider.Provider, error) {
	return &Adapter{client: c}, nil
}

func (a *Adapter) Name() string {
	return "cloudflare"
}

func (a *Adapter) doRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	config := a.client.GetConfig()
	url := "https://api.cloudflare.com/client/v4" + path
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}
	if config.APIUser != "" {
		req.Header.Set("X-Auth-Email", config.APIUser)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cloudflare api error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

func (a *Adapter) ListDomains(ctx context.Context) ([]model.Domain, error) {
	config := a.client.GetConfig()
	respBody, err := a.doRequest(ctx, "GET", "/zones", nil)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Result []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	domains := make([]model.Domain, 0, len(result.Result))
	for _, d := range result.Result {
		domains = append(domains, model.Domain{
			Name:       d.Name,
			User:       config.APIUser,
			IsExpired:  false,
			IsLocked:   false,
			AutoRenew:  true,
		})
	}
	
	return domains, nil
}

func (a *Adapter) GetDomainInfo(ctx context.Context, domainName string) (*model.Domain, error) {
	config := a.client.GetConfig()
	// First get the zone ID for the domain
	respBody, err := a.doRequest(ctx, "GET", fmt.Sprintf("/zones?name=%s", domainName), nil)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Result []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if len(result.Result) == 0 {
		return nil, fmt.Errorf("domain not found")
	}
	
	d := result.Result[0]
	return &model.Domain{
		Name:       d.Name,
		User:       config.APIUser,
		IsExpired:  false,
		IsLocked:   false,
		AutoRenew:  true,
	}, nil
}

func (a *Adapter) CheckAvailability(ctx context.Context, domainName string) (bool, error) {
	return false, errors.New("domain.CheckAvailability unsupported by cloudflare")
}

func (a *Adapter) RegisterDomain(ctx context.Context, req model.RegistrationRequest) error {
	return errors.New("domain.RegisterDomain unsupported by cloudflare")
}

func (a *Adapter) RenewDomain(ctx context.Context, domainName string, years int) error {
	return errors.New("domain.RenewDomain unsupported by cloudflare")
}

func (a *Adapter) GetNameservers(ctx context.Context, domainName string) ([]string, error) {
	return nil, errors.New("domain.GetNameservers unsupported by cloudflare")
}

func (a *Adapter) SetNameservers(ctx context.Context, domainName string, nameservers []string) error {
	return errors.New("domain.SetNameservers unsupported by cloudflare")
}

func (a *Adapter) SetDefaultNameservers(ctx context.Context, domainName string) error {
	return errors.New("domain.SetDefaultNameservers unsupported by cloudflare")
}

func (a *Adapter) EnableDNSSEC(ctx context.Context, domainName string) error {
	return errors.New("domain.EnableDNSSEC unsupported by cloudflare")
}

func (a *Adapter) DisableDNSSEC(ctx context.Context, domainName string) error {
	return errors.New("domain.DisableDNSSEC unsupported by cloudflare")
}

func (a *Adapter) GetDNSSECStatus(ctx context.Context, domainName string) (bool, error) {
	return false, errors.New("domain.GetDNSSECStatus unsupported by cloudflare")
}
