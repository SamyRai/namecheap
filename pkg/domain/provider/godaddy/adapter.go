package godaddy

import (
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"zonekit/pkg/client"
	"zonekit/pkg/domain/model"
	"zonekit/pkg/domain/provider"
)

type Adapter struct {
	client *client.Client
}

func init() {
	provider.Register("godaddy", New)
}

func New(c *client.Client) (provider.Provider, error) {
	return &Adapter{client: c}, nil
}

func (a *Adapter) Name() string {
	return "godaddy"
}

func (a *Adapter) doRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	config := a.client.GetConfig()
	baseURL := "https://api.godaddy.com/v1"
	if config.UseSandbox {
		baseURL = "https://api.ote-godaddy.com/v1"
	}
	url := baseURL + path
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	
	authHeader := "sso-key " + config.APIKey
	if config.APIUser != "" && !strings.Contains(config.APIKey, ":") {
		authHeader = "sso-key " + config.APIUser + ":" + config.APIKey
	}
	
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
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
		return nil, fmt.Errorf("godaddy api error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

func (a *Adapter) ListDomains(ctx context.Context) ([]model.Domain, error) {
	config := a.client.GetConfig()
	respBody, err := a.doRequest(ctx, "GET", "/domains", nil)
	if err != nil {
		return nil, err
	}
	
	var result []struct {
		Domain    string `json:"domain"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expires"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	domains := make([]model.Domain, 0, len(result))
	for _, d := range result {
		var exp time.Time
		if d.ExpiresAt != "" {
			exp, _ = time.Parse(time.RFC3339, d.ExpiresAt)
		}
		domains = append(domains, model.Domain{
			Name:       d.Domain,
			User:       config.APIUser,
			Expires:    d.ExpiresAt,
			IsExpired:  exp.Before(time.Now()),
		})
	}
	
	return domains, nil
}

func (a *Adapter) GetDomainInfo(ctx context.Context, domainName string) (*model.Domain, error) {
	config := a.client.GetConfig()
	respBody, err := a.doRequest(ctx, "GET", fmt.Sprintf("/domains/%s", domainName), nil)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Domain    string `json:"domain"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expires"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var exp time.Time
	if result.ExpiresAt != "" {
		exp, _ = time.Parse(time.RFC3339, result.ExpiresAt)
	}
	
	return &model.Domain{
		Name:       result.Domain,
		User:       config.APIUser,
		Expires:    result.ExpiresAt,
		IsExpired:  exp.Before(time.Now()),
	}, nil
}

func (a *Adapter) CheckAvailability(ctx context.Context, domainName string) (bool, error) {
	respBody, err := a.doRequest(ctx, "GET", fmt.Sprintf("/domains/available?domain=%s", domainName), nil)
	if err != nil {
		return false, err
	}
	
	var result struct {
		Available bool `json:"available"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return result.Available, nil
}

func (a *Adapter) RegisterDomain(ctx context.Context, req model.RegistrationRequest) error {
	// Build minimal payload for GoDaddy
	payload := fmt.Sprintf(`{
		"domain": "%s",
		"consent": {
			"agreedAt": "%s",
			"agreedBy": "ZoneKit Client"
		},
		"period": %d,
		"nameServers": []
	}`, req.DomainName, time.Now().Format(time.RFC3339), req.Years)
	
	_, err := a.doRequest(ctx, "POST", "/domains/purchase", strings.NewReader(payload))
	return err
}

func (a *Adapter) RenewDomain(ctx context.Context, domainName string, years int) error {
	payload := fmt.Sprintf(`{
		"period": %d
	}`, years)
	
	_, err := a.doRequest(ctx, "POST", fmt.Sprintf("/domains/%s/renew", domainName), strings.NewReader(payload))
	return err
}

func (a *Adapter) GetNameservers(ctx context.Context, domainName string) ([]string, error) {
	return nil, errors.New("domain.GetNameservers unsupported by godaddy")
}

func (a *Adapter) SetNameservers(ctx context.Context, domainName string, nameservers []string) error {
	return errors.New("domain.SetNameservers unsupported by godaddy")
}

func (a *Adapter) SetDefaultNameservers(ctx context.Context, domainName string) error {
	return errors.New("domain.SetDefaultNameservers unsupported by godaddy")
}

func (a *Adapter) EnableDNSSEC(ctx context.Context, domainName string) error {
	return errors.New("domain.EnableDNSSEC unsupported by godaddy")
}

func (a *Adapter) DisableDNSSEC(ctx context.Context, domainName string) error {
	return errors.New("domain.DisableDNSSEC unsupported by godaddy")
}

func (a *Adapter) GetDNSSECStatus(ctx context.Context, domainName string) (bool, error) {
	return false, errors.New("domain.GetDNSSECStatus unsupported by godaddy")
}
