package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
	"zonekit/internal/testutil"
)

const (
	testCloudflareToken     = "test-token"
	testCloudflareAccountID = "test-account-id"
)

// ConfigTestSuite is a test suite for config package
type ConfigTestSuite struct {
	suite.Suite
	manager    *Manager
	configPath string
}

// TestConfigSuite runs the config test suite
func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	s.configPath = filepath.Join(tmpDir, "test-config.yaml")

	manager, err := NewManagerWithPath(s.configPath)
	s.Require().NoError(err)
	s.manager = manager
}

func (s *ConfigTestSuite) TestNewManagerWithPath_NewConfigFile() {
	manager, err := NewManagerWithPath(s.configPath)

	s.Require().NoError(err)
	s.Require().NotNil(manager)
	s.Require().Equal(s.configPath, manager.GetConfigPath())
}

func (s *ConfigTestSuite) TestNewManagerWithPath_ExistingConfigFile() {
	// Create a valid config file
	fixture := testutil.AccountConfigFixture()
	config := &Config{
		Accounts: map[string]*AccountConfig{
			"default": {
				Username:    fixture.Username,
				APIUser:     fixture.APIUser,
				APIKey:      fixture.APIKey,
				ClientIP:    fixture.ClientIP,
				UseSandbox:  fixture.UseSandbox,
				Description: fixture.Description,
			},
		},
		CurrentAccount: "default",
	}
	data, err := config.marshal()
	s.Require().NoError(err)

	err = os.WriteFile(s.configPath, data, 0600)
	s.Require().NoError(err)

	manager, err := NewManagerWithPath(s.configPath)

	s.Require().NoError(err)
	s.Require().NotNil(manager)
	s.Require().Equal(s.configPath, manager.GetConfigPath())
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_ValidAccount() {
	fixture := testutil.AccountConfigFixture()
	account := &AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().NoError(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_MissingUsername() {
	fixture := testutil.AccountConfigFixtureWithValues("", "test", "test", "127.0.0.1", true)
	account := &AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_MissingAPIUser() {
	fixture := testutil.AccountConfigFixtureWithValues("test", "", "test", "127.0.0.1", true)
	account := &AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_MissingAPIKey() {
	fixture := testutil.AccountConfigFixtureWithValues("test", "test", "", "127.0.0.1", true)
	account := &AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_MissingClientIP() {
	fixture := testutil.AccountConfigFixtureWithValues("test", "test", "test", "", true)
	account := &AccountConfig{
		Username:   fixture.Username,
		APIUser:    fixture.APIUser,
		APIKey:     fixture.APIKey,
		ClientIP:   fixture.ClientIP,
		UseSandbox: fixture.UseSandbox,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_CloudflareValid() {
	account := &AccountConfig{
		Provider:  ProviderCloudflare,
		APIToken:  testCloudflareToken,
		AccountID: testCloudflareAccountID,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().NoError(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_CloudflareUserScopeNoAccountID() {
	account := &AccountConfig{
		Provider:   ProviderCloudflare,
		APIToken:   testCloudflareToken,
		TokenScope: TokenScopeUser,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().NoError(err, "a user-scoped token does not need an account_id")
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_CloudflareMissingToken() {
	account := &AccountConfig{
		Provider:  ProviderCloudflare,
		AccountID: testCloudflareAccountID,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_CloudflareAccountScopeMissingAccountID() {
	account := &AccountConfig{
		Provider: ProviderCloudflare,
		APIToken: testCloudflareToken,
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err, "an account-scoped token (the default) requires account_id")
}

func (s *ConfigTestSuite) TestManager_ValidateAccount_CloudflareInvalidTokenScope() {
	account := &AccountConfig{
		Provider:   ProviderCloudflare,
		APIToken:   testCloudflareToken,
		AccountID:  testCloudflareAccountID,
		TokenScope: "bogus",
	}
	err := s.manager.ValidateAccount(account)
	s.Require().Error(err)
}

// TestNamecheapConfig_LoadsUnchanged loads a copy of the documented
// multi-account config format (see configs/config.example.yaml) and
// checks a Namecheap account with no "provider" key - the shape every
// config written before this package supported Cloudflare has - still
// loads with GetProvider() defaulting to "namecheap" and every field
// intact.
func (s *ConfigTestSuite) TestNamecheapConfig_LoadsUnchanged() {
	const legacyStyleConfig = `
accounts:
  default:
    username: "my-namecheap-username"
    api_user: "my-api-username"
    api_key: "my-api-key-here"
    client_ip: "203.0.113.5"
    use_sandbox: false
    description: "My main Namecheap account"
current_account: "default"
`
	err := os.WriteFile(s.configPath, []byte(legacyStyleConfig), 0600)
	s.Require().NoError(err)

	manager, err := NewManagerWithPath(s.configPath)
	s.Require().NoError(err)

	account, err := manager.GetCurrentAccount()
	s.Require().NoError(err)
	s.Require().Equal("namecheap", account.GetProvider())
	s.Require().Equal("my-namecheap-username", account.Username)
	s.Require().Equal("my-api-username", account.APIUser)
	s.Require().Equal("my-api-key-here", account.APIKey)
	s.Require().Equal("203.0.113.5", account.ClientIP)
	s.Require().NoError(manager.ValidateAccount(account))
}

// TestCloudflareConfig_Loads loads a Cloudflare account written in the
// same documented multi-account format, confirming the two provider
// shapes coexist under one Config without interfering with each other.
func (s *ConfigTestSuite) TestCloudflareConfig_Loads() {
	const cfg = `
accounts:
  default:
    provider: "namecheap"
    username: "my-namecheap-username"
    api_user: "my-api-username"
    api_key: "my-api-key-here"
    client_ip: "203.0.113.5"
  cloudflare:
    provider: "cloudflare"
    api_token: "my-cloudflare-token"
    account_id: "abc123"
    description: "Cloudflare account"
current_account: "default"
`
	err := os.WriteFile(s.configPath, []byte(cfg), 0600)
	s.Require().NoError(err)

	manager, err := NewManagerWithPath(s.configPath)
	s.Require().NoError(err)

	cf, err := manager.GetAccount("cloudflare")
	s.Require().NoError(err)
	s.Require().Equal("cloudflare", cf.GetProvider())
	s.Require().Equal("my-cloudflare-token", cf.APIToken)
	s.Require().Equal("abc123", cf.AccountID)
	s.Require().Empty(cf.Username, "a Cloudflare account must not require Namecheap fields")
	s.Require().NoError(manager.ValidateAccount(cf))

	nc, err := manager.GetAccount("default")
	s.Require().NoError(err)
	s.Require().Equal("namecheap", nc.GetProvider())
	s.Require().NoError(manager.ValidateAccount(nc))
}

func (s *ConfigTestSuite) TestManager_AddAccount() {
	fixture := testutil.AccountConfigFixture()
	account := &AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}

	// Test adding first account (manager creates default account, so we'll have 2)
	err := s.manager.AddAccount("test-account", account)
	s.Require().NoError(err)

	// Test adding duplicate account
	err = s.manager.AddAccount("test-account", account)
	s.Require().Error(err)

	// Verify account was added (default + test-account = 2)
	accounts := s.manager.ListAccounts()
	s.Require().GreaterOrEqual(len(accounts), 2)
	s.Require().Contains(accounts, "test-account")
}

func (s *ConfigTestSuite) TestManager_GetAccount() {
	fixture := testutil.AccountConfigFixture()
	account := &AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}

	err := s.manager.AddAccount("test-account", account)
	s.Require().NoError(err)

	// Test getting existing account
	got, err := s.manager.GetAccount("test-account")
	s.Require().NoError(err)
	s.Require().Equal(account.Username, got.Username)

	// Test getting non-existent account
	_, err = s.manager.GetAccount("non-existent")
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_RemoveAccount() {
	fixture := testutil.AccountConfigFixture()
	account := &AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}

	// Add two accounts (manager already has default, so we'll have 3 total)
	err := s.manager.AddAccount("account1", account)
	s.Require().NoError(err)

	err = s.manager.AddAccount("account2", account)
	s.Require().NoError(err)

	// Test removing account
	err = s.manager.RemoveAccount("account1")
	s.Require().NoError(err)

	// Remove account2 (should succeed, default still exists)
	err = s.manager.RemoveAccount("account2")
	s.Require().NoError(err)

	// Now test removing the last account (should fail)
	err = s.manager.RemoveAccount("default")
	s.Require().Error(err)

	// Test removing non-existent account
	err = s.manager.RemoveAccount("non-existent")
	s.Require().Error(err)
}

func (s *ConfigTestSuite) TestManager_SetCurrentAccount() {
	fixture := testutil.AccountConfigFixture()
	account := &AccountConfig{
		Username:    fixture.Username,
		APIUser:     fixture.APIUser,
		APIKey:      fixture.APIKey,
		ClientIP:    fixture.ClientIP,
		UseSandbox:  fixture.UseSandbox,
		Description: fixture.Description,
	}

	err := s.manager.AddAccount("account1", account)
	s.Require().NoError(err)

	err = s.manager.AddAccount("account2", account)
	s.Require().NoError(err)

	// Test setting current account
	err = s.manager.SetCurrentAccount("account2")
	s.Require().NoError(err)
	s.Require().Equal("account2", s.manager.GetCurrentAccountName())

	// Test setting non-existent account
	err = s.manager.SetCurrentAccount("non-existent")
	s.Require().Error(err)
}

// Helper method for testing
func (c *Config) marshal() ([]byte, error) {
	return yaml.Marshal(c)
}
