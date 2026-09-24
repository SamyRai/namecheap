package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Provider name constants for AccountConfig.Provider / GetProvider().
const (
	ProviderNamecheap  = "namecheap"
	ProviderCloudflare = "cloudflare"
)

// Cloudflare token-scope values for AccountConfig.TokenScope. They mirror
// which Cloudflare token-verification endpoint the token was minted
// against: an account-owned API token verifies at
// /accounts/{account_id}/tokens/verify (the common case - 15 of our 17
// zones use one), a user-owned token verifies at /user/tokens/verify.
const (
	TokenScopeAccount = "account"
	TokenScopeUser    = "user"
)

// AccountConfig represents a single DNS provider account configuration.
//
// Fields are shared across provider types rather than nested per-provider
// so that existing Namecheap configs (username/api_user/api_key/client_ip)
// keep loading unchanged: Provider defaults to "namecheap" when absent.
// Cloudflare accounts use a disjoint set of fields (APIToken, AccountID,
// TokenScope) and leave the Namecheap-only fields empty.
type AccountConfig struct {
	// Provider specifies which DNS provider to use (e.g., "namecheap", "cloudflare")
	// Defaults to "namecheap" if not specified for backward compatibility
	Provider    string `yaml:"provider,omitempty" mapstructure:"provider,omitempty"`
	Description string `yaml:"description,omitempty" mapstructure:"description,omitempty"`

	// Namecheap fields.
	Username   string `yaml:"username,omitempty" mapstructure:"username,omitempty"`
	APIUser    string `yaml:"api_user,omitempty" mapstructure:"api_user,omitempty"`
	APIKey     string `yaml:"api_key,omitempty" mapstructure:"api_key,omitempty"`
	ClientIP   string `yaml:"client_ip,omitempty" mapstructure:"client_ip,omitempty"`
	UseSandbox bool   `yaml:"use_sandbox,omitempty" mapstructure:"use_sandbox,omitempty"`

	// Cloudflare fields.
	// APIToken is a Cloudflare API token (Bearer auth).
	APIToken string `yaml:"api_token,omitempty" mapstructure:"api_token,omitempty"`
	// AccountID is the Cloudflare account ID. Required unless TokenScope
	// is "user".
	AccountID string `yaml:"account_id,omitempty" mapstructure:"account_id,omitempty"`
	// TokenScope selects the token-verification endpoint: "account"
	// (default) or "user". See the TokenScope* constants.
	TokenScope string `yaml:"token_scope,omitempty" mapstructure:"token_scope,omitempty"`
}

// Config represents the complete configuration structure
type Config struct {
	Accounts       map[string]*AccountConfig `yaml:"accounts" mapstructure:"accounts"`
	CurrentAccount string                    `yaml:"current_account" mapstructure:"current_account"`

	// Legacy fields for backward compatibility
	Username   string `yaml:"username" mapstructure:"username"`
	APIUser    string `yaml:"api_user" mapstructure:"api_user"`
	APIKey     string `yaml:"api_key" mapstructure:"api_key"`
	ClientIP   string `yaml:"client_ip" mapstructure:"client_ip"`
	UseSandbox bool   `yaml:"use_sandbox" mapstructure:"use_sandbox"`
}

// Manager handles configuration operations
type Manager struct {
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	// First try to find config in project directory
	projectConfigPath := FindProjectConfigPath()

	// Fall back to home directory if project config not found
	homeConfigPath := findHomeConfigPath()

	// Determine which config to use
	var configPath string
	if projectConfigPath != "" {
		configPath = projectConfigPath
	} else {
		configPath = homeConfigPath
	}

	return NewManagerWithPath(configPath)
}

// NewManagerWithPath creates a new configuration manager with a specific config path
func NewManagerWithPath(configPath string) (*Manager, error) {
	manager := &Manager{
		configPath: configPath,
		config:     &Config{},
	}

	// Load existing configuration if it exists
	if err := manager.Load(); err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			manager.config = manager.createDefaultConfig()
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Migrate legacy config if needed
	if err := manager.migrateLegacyConfig(); err != nil {
		return nil, fmt.Errorf("failed to migrate legacy config: %w", err)
	}

	return manager, nil
}

// findHomeConfigPath returns the home directory config path
func findHomeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".zonekit.yaml")
}

// Load reads the configuration from file
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, m.config)
}

// Save writes the configuration to file
func (m *Manager) Save() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	err = os.WriteFile(m.configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentAccount returns the currently selected account configuration
func (m *Manager) GetCurrentAccount() (*AccountConfig, error) {
	if m.config.CurrentAccount == "" {
		m.config.CurrentAccount = "default"
	}

	account, exists := m.config.Accounts[m.config.CurrentAccount]
	if !exists {
		return nil, fmt.Errorf("current account '%s' not found", m.config.CurrentAccount)
	}

	return account, nil
}

// GetAccount returns a specific account by name
func (m *Manager) GetAccount(name string) (*AccountConfig, error) {
	account, exists := m.config.Accounts[name]
	if !exists {
		return nil, fmt.Errorf("account '%s' not found", name)
	}

	return account, nil
}

// SetCurrentAccount changes the currently selected account
func (m *Manager) SetCurrentAccount(name string) error {
	if _, exists := m.config.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' not found", name)
	}

	m.config.CurrentAccount = name
	return m.Save()
}

// AddAccount adds a new account configuration
func (m *Manager) AddAccount(name string, account *AccountConfig) error {
	if m.config.Accounts == nil {
		m.config.Accounts = make(map[string]*AccountConfig)
	}

	if _, exists := m.config.Accounts[name]; exists {
		return fmt.Errorf("account '%s' already exists", name)
	}

	m.config.Accounts[name] = account

	// Set as current if it's the first account
	if len(m.config.Accounts) == 1 {
		m.config.CurrentAccount = name
	}

	return m.Save()
}

// UpdateAccount updates an existing account configuration
func (m *Manager) UpdateAccount(name string, account *AccountConfig) error {
	if m.config.Accounts == nil {
		return fmt.Errorf("no accounts configured")
	}

	if _, exists := m.config.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' not found", name)
	}

	m.config.Accounts[name] = account
	return m.Save()
}

// RemoveAccount removes an account configuration
func (m *Manager) RemoveAccount(name string) error {
	if m.config.Accounts == nil {
		return fmt.Errorf("no accounts configured")
	}

	if _, exists := m.config.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' not found", name)
	}

	// Don't allow removing the last account
	if len(m.config.Accounts) == 1 {
		return fmt.Errorf("cannot remove the last account")
	}

	// If removing current account, switch to another one
	if m.config.CurrentAccount == name {
		for accountName := range m.config.Accounts {
			if accountName != name {
				m.config.CurrentAccount = accountName
				break
			}
		}
	}

	delete(m.config.Accounts, name)
	return m.Save()
}

// ListAccounts returns all account names
func (m *Manager) ListAccounts() []string {
	if m.config.Accounts == nil {
		return []string{}
	}

	accounts := make([]string, 0, len(m.config.Accounts))
	for name := range m.config.Accounts {
		accounts = append(accounts, name)
	}

	return accounts
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetConfigLocation returns a human-readable description of where the config is located
func (m *Manager) GetConfigLocation() string {
	if filepath.Dir(m.configPath) == filepath.Join(os.Getenv("HOME"), "configs") {
		return "project directory (configs/.zonekit.yaml)"
	}
	return "home directory (~/.zonekit.yaml)"
}

// GetCurrentAccountName returns the name of the currently selected account
func (m *Manager) GetCurrentAccountName() string {
	return m.config.CurrentAccount
}

// createDefaultConfig creates a default configuration structure
func (m *Manager) createDefaultConfig() *Config {
	return &Config{
		Accounts: map[string]*AccountConfig{
			"default": {
				Provider:    ProviderNamecheap, // Default provider for backward compatibility
				Username:    "your-provider-username",
				APIUser:     "your-api-username",
				APIKey:      "your-api-key-here",
				ClientIP:    "your.public.ip.address",
				UseSandbox:  false,
				Description: "Default account",
			},
		},
		CurrentAccount: "default",
	}
}

// migrateLegacyConfig migrates legacy single-account configuration to new format
func (m *Manager) migrateLegacyConfig() error {
	// Check if we need to migrate (legacy fields exist and no accounts configured)
	if (m.config.Username != "" || m.config.APIUser != "" || m.config.APIKey != "" || m.config.ClientIP != "") &&
		(m.config.Accounts == nil || len(m.config.Accounts) == 0) {

		// Create default account from legacy fields
		defaultAccount := &AccountConfig{
			Provider:    ProviderNamecheap, // Default provider for migrated legacy configs
			Username:    m.config.Username,
			APIUser:     m.config.APIUser,
			APIKey:      m.config.APIKey,
			ClientIP:    m.config.ClientIP,
			UseSandbox:  m.config.UseSandbox,
			Description: "Migrated from legacy configuration",
		}

		// Initialize accounts map if needed
		if m.config.Accounts == nil {
			m.config.Accounts = make(map[string]*AccountConfig)
		}

		// Add the migrated account
		m.config.Accounts["default"] = defaultAccount
		m.config.CurrentAccount = "default"

		// Clear legacy fields
		m.config.Username = ""
		m.config.APIUser = ""
		m.config.APIKey = ""
		m.config.ClientIP = ""
		m.config.UseSandbox = false

		// Save the migrated configuration
		if err := m.Save(); err != nil {
			return fmt.Errorf("failed to save migrated config: %w", err)
		}
	}

	return nil
}

// GetProvider returns the provider name for an account, defaulting to "namecheap" for backward compatibility
func (a *AccountConfig) GetProvider() string {
	if a.Provider == "" {
		return ProviderNamecheap // Default provider for backward compatibility
	}
	return a.Provider
}

// ValidateAccount validates an account configuration for its provider.
func (m *Manager) ValidateAccount(account *AccountConfig) error {
	switch account.GetProvider() {
	case ProviderCloudflare:
		return validateCloudflareAccount(account)
	default:
		return validateNamecheapAccount(account)
	}
}

func validateNamecheapAccount(account *AccountConfig) error {
	if account.Username == "" {
		return fmt.Errorf("username is required")
	}
	if account.APIUser == "" {
		return fmt.Errorf("api_user is required")
	}
	if account.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if account.ClientIP == "" {
		return fmt.Errorf("client_ip is required")
	}
	return nil
}

func validateCloudflareAccount(account *AccountConfig) error {
	if account.APIToken == "" {
		return fmt.Errorf("api_token is required")
	}

	scope := account.TokenScope
	if scope == "" {
		scope = TokenScopeAccount
	}
	if scope != TokenScopeAccount && scope != TokenScopeUser {
		return fmt.Errorf("token_scope must be %q or %q", TokenScopeAccount, TokenScopeUser)
	}
	if scope == TokenScopeAccount && account.AccountID == "" {
		return fmt.Errorf("account_id is required unless token_scope is %q", TokenScopeUser)
	}
	return nil
}
