package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AccountConfig represents a single DNS provider account configuration
type AccountConfig struct {
	// Provider specifies which DNS provider to use (e.g., "namecheap", "cloudflare")
	// Defaults to "namecheap" if not specified for backward compatibility
	Provider    string `yaml:"provider,omitempty" mapstructure:"provider,omitempty"`
	Username    string `yaml:"username" mapstructure:"username"`
	APIUser     string `yaml:"api_user" mapstructure:"api_user"`
	APIKey      string `yaml:"api_key" mapstructure:"api_key"`
	ClientIP    string `yaml:"client_ip" mapstructure:"client_ip"`
	UseSandbox  bool   `yaml:"use_sandbox" mapstructure:"use_sandbox"`
	Description string `yaml:"description" mapstructure:"description"`
}

// Config represents the complete configuration structure
type Config struct {
	Accounts       map[string]*AccountConfig `yaml:"accounts" mapstructure:"accounts"`
	CurrentAccount string                    `yaml:"current_account" mapstructure:"current_account"`
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
