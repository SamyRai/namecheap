package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.glpx.pro/zonekit/pkg/config"
)

// accountAddProvider backs the `account add --provider` flag.
var accountAddProvider string

// accountCmd represents the account command
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage multiple DNS provider accounts",
	Long:  `Commands for managing multiple DNS provider account configurations.`,
}

// accountListCmd represents the account list command
var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured accounts",
	Long:  `Display all configured DNS provider accounts and show which one is currently active.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("No accounts configured.")
			fmt.Println("Run 'zonekit account add' to add your first account.")
			return nil
		}

		// Validate that we can access current account
		if _, err := configManager.GetCurrentAccount(); err != nil {
			return fmt.Errorf("failed to get current account: %w", err)
		}

		fmt.Println("Configured Accounts:")
		fmt.Println("====================")
		fmt.Println()

		for _, accountName := range accounts {
			account, err := configManager.GetAccount(accountName)
			if err != nil {
				fmt.Printf("⚠️  %s: Error loading account details\n", accountName)
				continue
			}
			printAccountSummary(accountName, account, accountName == configManager.GetCurrentAccountName())
		}

		return nil
	},
}

// printAccountSummary prints the account list's per-account block. Field
// selection depends on the account's provider: Namecheap accounts show
// username/API user/client IP, Cloudflare accounts show the masked API
// token and account ID.
func printAccountSummary(name string, account *config.AccountConfig, isCurrent bool) {
	if isCurrent {
		fmt.Printf("→ %s (current)\n", name)
	} else {
		fmt.Printf("  %s\n", name)
	}

	fmt.Printf("   Provider: %s\n", account.GetProvider())
	switch account.GetProvider() {
	case config.ProviderCloudflare:
		fmt.Printf("   API Token: %s\n", config.MaskAPIKey(account.APIToken))
		fmt.Printf("   Account ID: %s\n", valueOrNotSet(account.AccountID))
		fmt.Printf("   Token Scope: %s\n", tokenScopeOrDefault(account.TokenScope))
	default:
		fmt.Printf("   Username: %s\n", account.Username)
		fmt.Printf("   API User: %s\n", account.APIUser)
		fmt.Printf("   Client IP: %s\n", account.ClientIP)
		fmt.Printf("   Sandbox: %t\n", account.UseSandbox)
	}
	if account.Description != "" {
		fmt.Printf("   Description: %s\n", account.Description)
	}
	fmt.Println()
}

func valueOrNotSet(v string) string {
	if v == "" {
		return "(not set)"
	}
	return v
}

func tokenScopeOrDefault(scope string) string {
	if scope == "" {
		return config.TokenScopeAccount
	}
	return scope
}

// accountAddCmd represents the account add command
var accountAddCmd = &cobra.Command{
	Use:   "add [account-name]",
	Short: "Add a new account configuration",
	Long: `Add a new DNS provider account configuration with an interactive prompt.

Use --provider to select the provider without an interactive prompt:

  zonekit account add cloudflare --provider cloudflare
  zonekit account add work --provider namecheap`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		// Get account name
		accountName := "default"
		if len(args) > 0 {
			accountName = args[0]
		}

		// Check if account already exists
		if _, err := configManager.GetAccount(accountName); err == nil {
			return fmt.Errorf("account '%s' already exists", accountName)
		}

		providerName, err := resolveProviderChoice(accountAddProvider)
		if err != nil {
			return err
		}

		fmt.Printf("Adding new account: %s (provider: %s)\n", accountName, providerName)
		fmt.Println("================================")
		fmt.Println()

		var account *config.AccountConfig
		switch providerName {
		case config.ProviderCloudflare:
			account = promptCloudflareAccount(&config.AccountConfig{})
		default:
			account = promptNamecheapAccount(&config.AccountConfig{})
		}
		account.Provider = providerName

		// Validate account
		if err := configManager.ValidateAccount(account); err != nil {
			return fmt.Errorf("invalid account configuration: %w", err)
		}

		// Add account
		if err := configManager.AddAccount(accountName, account); err != nil {
			return fmt.Errorf("failed to add account: %w", err)
		}

		fmt.Printf("✅ Account '%s' added successfully!\n", accountName)

		// Ask if user wants to switch to this account
		var switchInput string
		fmt.Printf("Switch to account '%s'? (Y/n): ", accountName)
		_, _ = fmt.Scanln(&switchInput)
		if switchInput == "" || strings.ToLower(switchInput) == "y" || strings.ToLower(switchInput) == "yes" {
			if err := configManager.SetCurrentAccount(accountName); err != nil {
				return fmt.Errorf("failed to switch to account '%s': %w", accountName, err)
			}
			fmt.Printf("✅ Switched to account '%s'\n", accountName)
		}

		return nil
	},
}

// resolveProviderChoice returns the flag value if set, otherwise prompts
// interactively, defaulting to Namecheap for backward compatibility.
func resolveProviderChoice(flagValue string) (string, error) {
	providerName := strings.ToLower(strings.TrimSpace(flagValue))
	if providerName == "" {
		fmt.Printf("Provider (%s/%s) [%s]: ", config.ProviderNamecheap, config.ProviderCloudflare, config.ProviderNamecheap)
		var input string
		_, _ = fmt.Scanln(&input)
		providerName = strings.ToLower(strings.TrimSpace(input))
		if providerName == "" {
			providerName = config.ProviderNamecheap
		}
	}

	if providerName != config.ProviderNamecheap && providerName != config.ProviderCloudflare {
		return "", fmt.Errorf("unsupported provider %q (must be %q or %q)", providerName, config.ProviderNamecheap, config.ProviderCloudflare)
	}
	return providerName, nil
}

// promptNamecheapAccount interactively fills the Namecheap fields of an
// account, using existing as the field defaults (so it doubles as the
// edit-command prompt).
func promptNamecheapAccount(existing *config.AccountConfig) *config.AccountConfig {
	account := &config.AccountConfig{}

	account.Username = promptWithDefault("Provider Username", existing.Username)
	account.APIUser = promptWithDefault("API User", existing.APIUser)
	account.APIKey = promptWithDefault(fmt.Sprintf("API Key [%s***]", maskedPrefix(existing.APIKey)), "")
	if account.APIKey == "" {
		account.APIKey = existing.APIKey
	}
	account.ClientIP = promptWithDefault("Client IP Address", existing.ClientIP)

	sandboxInput := promptWithDefault("Use Sandbox Environment? (y/N)", "")
	if sandboxInput == "" {
		account.UseSandbox = existing.UseSandbox
	} else {
		account.UseSandbox = strings.EqualFold(sandboxInput, "y") || strings.EqualFold(sandboxInput, "yes")
	}

	account.Description = promptWithDefault("Description (optional)", existing.Description)
	return account
}

// promptCloudflareAccount interactively fills the Cloudflare fields of
// an account, using existing as the field defaults.
func promptCloudflareAccount(existing *config.AccountConfig) *config.AccountConfig {
	account := &config.AccountConfig{}

	account.APIToken = promptWithDefault(fmt.Sprintf("Cloudflare API Token [%s***]", maskedPrefix(existing.APIToken)), "")
	if account.APIToken == "" {
		account.APIToken = existing.APIToken
	}

	scopeDefault := tokenScopeOrDefault(existing.TokenScope)
	scopeInput := promptWithDefault(fmt.Sprintf("Token Scope (%s/%s)", config.TokenScopeAccount, config.TokenScopeUser), scopeDefault)
	account.TokenScope = strings.ToLower(strings.TrimSpace(scopeInput))

	if account.TokenScope == config.TokenScopeAccount {
		account.AccountID = promptWithDefault("Cloudflare Account ID", existing.AccountID)
	} else {
		account.AccountID = existing.AccountID
	}

	account.Description = promptWithDefault("Description (optional)", existing.Description)
	return account
}

// promptWithDefault prints "label [default]: ", reads one line, and
// returns the typed value or default when the line is empty.
func promptWithDefault(label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	var input string
	_, _ = fmt.Scanln(&input)
	if input == "" {
		return defaultValue
	}
	return input
}

func maskedPrefix(secret string) string {
	if len(secret) > 4 {
		return secret[:4]
	}
	return secret
}

// accountSwitchCmd represents the account switch command
var accountSwitchCmd = &cobra.Command{
	Use:   "switch [account-name]",
	Short: "Switch to a different account",
	Long:  `Switch to a different configured DNS provider account.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]

		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		// Check if account exists
		if _, err := configManager.GetAccount(accountName); err != nil {
			return fmt.Errorf("account '%s' not found: %w", accountName, err)
		}

		if accountName == configManager.GetCurrentAccountName() {
			fmt.Printf("Already using account '%s'\n", accountName)
			return nil
		}

		previousName := configManager.GetCurrentAccountName()

		// Switch account
		if err := configManager.SetCurrentAccount(accountName); err != nil {
			return fmt.Errorf("failed to switch to account '%s': %w", accountName, err)
		}

		fmt.Printf("✅ Switched from account '%s' to '%s'\n", previousName, accountName)
		return nil
	},
}

// accountRemoveCmd represents the account remove command
var accountRemoveCmd = &cobra.Command{
	Use:   "remove [account-name]",
	Short: "Remove an account configuration",
	Long:  `Remove a DNS provider account configuration. Cannot remove the last remaining account.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]

		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		// Check if account exists
		if _, err := configManager.GetAccount(accountName); err != nil {
			return fmt.Errorf("account '%s' not found: %w", accountName, err)
		}

		wasCurrent := accountName == configManager.GetCurrentAccountName()

		// Confirm removal
		fmt.Printf("Are you sure you want to remove account '%s'? (y/N): ", accountName)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Aborted.")
			return nil
		}

		// Remove account
		if err := configManager.RemoveAccount(accountName); err != nil {
			return fmt.Errorf("failed to remove account '%s': %w", accountName, err)
		}

		fmt.Printf("✅ Account '%s' removed successfully!\n", accountName)

		// Show new current account if it changed
		if wasCurrent {
			if newCurrent := configManager.GetCurrentAccountName(); newCurrent != "" {
				fmt.Printf("Switched to account '%s'\n", newCurrent)
			}
		}

		return nil
	},
}

// accountShowCmd represents the account show command
var accountShowCmd = &cobra.Command{
	Use:   "show [account-name]",
	Short: "Show details of a specific account",
	Long:  `Display detailed information about a specific DNS provider account configuration.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		// Determine which account to show
		var accountName string
		if len(args) > 0 {
			accountName = args[0]
		} else {
			accountName = configManager.GetCurrentAccountName()
		}

		// Get account
		account, err := configManager.GetAccount(accountName)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", accountName, err)
		}

		// Display account details
		fmt.Printf("Account: %s\n", accountName)
		if accountName == configManager.GetCurrentAccountName() {
			fmt.Println("Status: Current (active)")
		} else {
			fmt.Println("Status: Inactive")
		}
		fmt.Println("========================")
		fmt.Println()

		fmt.Printf("Provider: %s\n", account.GetProvider())
		switch account.GetProvider() {
		case config.ProviderCloudflare:
			fmt.Printf("API Token: %s\n", config.MaskAPIKey(account.APIToken))
			fmt.Printf("Account ID: %s\n", valueOrNotSet(account.AccountID))
			fmt.Printf("Token Scope: %s\n", tokenScopeOrDefault(account.TokenScope))
		default:
			fmt.Printf("Username: %s\n", account.Username)
			fmt.Printf("API User: %s\n", account.APIUser)
			fmt.Printf("API Key: %s\n", config.MaskAPIKey(account.APIKey))
			fmt.Printf("Client IP: %s\n", account.ClientIP)
			fmt.Printf("Sandbox: %t\n", account.UseSandbox)
		}
		if account.Description != "" {
			fmt.Printf("Description: %s\n", account.Description)
		}

		return nil
	},
}

// accountEditCmd represents the account edit command
var accountEditCmd = &cobra.Command{
	Use:   "edit [account-name]",
	Short: "Edit an existing account configuration",
	Long: `Edit an existing DNS provider account configuration with an interactive prompt.

The account's provider cannot be changed by editing; remove and re-add the
account with a different --provider instead.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		// Determine which account to edit
		var accountName string
		if len(args) > 0 {
			accountName = args[0]
		} else {
			accountName = configManager.GetCurrentAccountName()
		}

		// Get existing account
		existingAccount, err := configManager.GetAccount(accountName)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", accountName, err)
		}

		fmt.Printf("Editing account: %s (provider: %s)\n", accountName, existingAccount.GetProvider())
		fmt.Println("================================")
		fmt.Println()

		var account *config.AccountConfig
		switch existingAccount.GetProvider() {
		case config.ProviderCloudflare:
			account = promptCloudflareAccount(existingAccount)
		default:
			account = promptNamecheapAccount(existingAccount)
		}
		account.Provider = existingAccount.GetProvider()

		// Validate account
		if err := configManager.ValidateAccount(account); err != nil {
			return fmt.Errorf("invalid account configuration: %w", err)
		}

		// Update account
		if err := configManager.UpdateAccount(accountName, account); err != nil {
			return fmt.Errorf("failed to update account: %w", err)
		}

		fmt.Printf("✅ Account '%s' updated successfully!\n", accountName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountShowCmd)
	accountCmd.AddCommand(accountEditCmd)

	accountAddCmd.Flags().StringVar(&accountAddProvider, "provider", "", "DNS provider for this account (namecheap or cloudflare)")
}
