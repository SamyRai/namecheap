package client

import (
	"fmt"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"zonekit/pkg/config"
)

// Client wraps the Namecheap SDK client with additional functionality
type Client struct {
	nc     *namecheap.Client
	config *config.AccountConfig
}

// NewClient creates a new Namecheap client with the given configuration
func NewClient(accountConfig *config.AccountConfig) (*Client, error) {
	// Basic validation
	if accountConfig.Username == "" || accountConfig.APIUser == "" || accountConfig.APIKey == "" || accountConfig.ClientIP == "" {
		return nil, fmt.Errorf("invalid configuration: missing required fields")
	}

	nc := namecheap.NewClient(&namecheap.ClientOptions{
		UserName:   accountConfig.Username,
		ApiUser:    accountConfig.APIUser,
		ApiKey:     accountConfig.APIKey,
		ClientIp:   accountConfig.ClientIP,
		UseSandbox: accountConfig.UseSandbox,
	})

	return &Client{
		nc:     nc,
		config: accountConfig,
	}, nil
}

// NewClientFromConfig creates a new client using the configuration manager
func NewClientFromConfig(configManager *config.Manager) (*Client, error) {
	accountConfig, err := configManager.GetCurrentAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to get current account: %w", err)
	}

	return NewClient(accountConfig)
}

// NewClientForAccount creates a new client for a specific account
func NewClientForAccount(configManager *config.Manager, accountName string) (*Client, error) {
	accountConfig, err := configManager.GetAccount(accountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get account '%s': %w", accountName, err)
	}

	return NewClient(accountConfig)
}

// GetNamecheapClient returns the underlying Namecheap SDK client
func (c *Client) GetNamecheapClient() *namecheap.Client {
	return c.nc
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *config.AccountConfig {
	return c.config
}

// GetAccountName returns the name of the account this client is using
func (c *Client) GetAccountName() string {
	// This would need to be passed in or stored if we want to track the account name
	// For now, return the username as a reasonable identifier
	return c.config.Username
}

// NewClientFromViper creates a new client using viper configuration
func NewClientFromViper() (*Client, error) {
	configManager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}
	return NewClientFromConfig(configManager)
}
