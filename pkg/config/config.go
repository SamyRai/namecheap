package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

// AccountConfig represents a single DNS provider account configuration
type AccountConfig struct {
	// Provider specifies which DNS provider to use (e.g., "namecheap", "cloudflare")
	// Defaults to "namecheap" if not specified for backward compatibility
	Provider    string `yaml:"provider,omitempty" mapstructure:"provider,omitempty"`
	Username    string `yaml:"username" mapstructure:"username"`
	APIUser     string `yaml:"api_user,omitempty" mapstructure:"api_user,omitempty"`
	APIKey      string `yaml:"api_key,omitempty" mapstructure:"api_key,omitempty"`
	ClientIP    string `yaml:"client_ip,omitempty" mapstructure:"client_ip,omitempty"`
	UseSandbox  bool   `yaml:"use_sandbox" mapstructure:"use_sandbox"`
	Description string `yaml:"description" mapstructure:"description"`
}

// ContactProfile represents a contact profile for domain registration
type ContactProfile struct {
	FirstName      string `yaml:"first_name" mapstructure:"first_name"`
	LastName       string `yaml:"last_name" mapstructure:"last_name"`
	Address        string `yaml:"address" mapstructure:"address"`
	City           string `yaml:"city" mapstructure:"city"`
	StateProvince  string `yaml:"state_province" mapstructure:"state_province"`
	PostalCode     string `yaml:"postal_code" mapstructure:"postal_code"`
	Country        string `yaml:"country" mapstructure:"country"`
	Phone          string `yaml:"phone" mapstructure:"phone"`
	Email          string `yaml:"email" mapstructure:"email"`
}

// Config represents the complete configuration structure
type Config struct {
	Accounts        map[string]*AccountConfig  `yaml:"accounts" mapstructure:"accounts"`
	CurrentAccount  string                     `yaml:"current_account" mapstructure:"current_account"`
	ContactProfiles map[string]*ContactProfile `yaml:"contact_profiles,omitempty" mapstructure:"contact_profiles,omitempty"`
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

	if err := yaml.Unmarshal(data, m.config); err != nil {
		return err
	}

	// Try to load secrets from keyring for all accounts
	for name, account := range m.config.Accounts {
		m.LoadSecretsFromKeyring(name, account)
		// Check environment variables as fallback
		if account.APIKey == "" {
			account.APIKey = os.Getenv("ZONEKIT_" + strings.ToUpper(name) + "_API_KEY")
		}
		if account.APIUser == "" {
			account.APIUser = os.Getenv("ZONEKIT_" + strings.ToUpper(name) + "_API_USER")
		}
	}

	return nil
}

// LoadSecretsFromKeyring populates account credentials from OS keyring if available
func (m *Manager) LoadSecretsFromKeyring(name string, account *AccountConfig) {
	key, err := keyring.Get("zonekit", name+"_api_key")
	if err == nil && key != "" {
		account.APIKey = key
	}
	user, err := keyring.Get("zonekit", name+"_api_user")
	if err == nil && user != "" {
		account.APIUser = user
	}
}

// SaveSecretsToKeyring saves account credentials to OS keyring and removes them from account config to avoid writing to yaml
func (m *Manager) SaveSecretsToKeyring(name string, account *AccountConfig) error {
	if account.APIKey != "" {
		err := keyring.Set("zonekit", name+"_api_key", account.APIKey)
		if err != nil {
			return fmt.Errorf("failed to save API Key to keyring: %w", err)
		}
		// Clear it from yaml config
		account.APIKey = ""
	}
	if account.APIUser != "" {
		err := keyring.Set("zonekit", name+"_api_user", account.APIUser)
		if err != nil {
			return fmt.Errorf("failed to save API User to keyring: %w", err)
		}
		// Clear it from yaml config
		account.APIUser = ""
	}
	return nil
}

// Save writes the configuration to file
func (m *Manager) Save() error {
	// Ensure we don't marshal raw secrets if they are supposed to be in keyring
	// For backward compatibility, we marshal them if they are still in the struct
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

// GetContactProfile returns a specific contact profile by name
func (m *Manager) GetContactProfile(name string) (*ContactProfile, error) {
	if m.config.ContactProfiles == nil {
		return nil, fmt.Errorf("no contact profiles configured")
	}
	
	profile, exists := m.config.ContactProfiles[name]
	if !exists {
		return nil, fmt.Errorf("contact profile '%s' not found", name)
	}

	return profile, nil
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
				Provider:    "namecheap",
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

// GetProvider returns the provider name for an account, defaulting to "namecheap"
func (a *AccountConfig) GetProvider() string {
	if a.Provider == "" {
		return "namecheap"
	}
	return a.Provider
}

// ValidateAccount validates an account configuration
func (m *Manager) ValidateAccount(account *AccountConfig) error {
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
