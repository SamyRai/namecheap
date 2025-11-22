package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"namecheap-dns-manager/pkg/config"
)

// accountCmd represents the account command
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage multiple Namecheap accounts",
	Long:  `Commands for managing multiple Namecheap account configurations.`,
}

// accountListCmd represents the account list command
var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured accounts",
	Long:  `Display all configured Namecheap accounts and show which one is currently active.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create config manager: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("No accounts configured.")
			fmt.Println("Run 'namecheap-dns account add' to add your first account.")
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

			// Show current account indicator
			if accountName == configManager.GetCurrentAccountName() {
				fmt.Printf("→ %s (current)\n", accountName)
			} else {
				fmt.Printf("  %s\n", accountName)
			}

			fmt.Printf("   Username: %s\n", account.Username)
			fmt.Printf("   API User: %s\n", account.APIUser)
			fmt.Printf("   Client IP: %s\n", account.ClientIP)
			fmt.Printf("   Sandbox: %t\n", account.UseSandbox)
			if account.Description != "" {
				fmt.Printf("   Description: %s\n", account.Description)
			}
			fmt.Println()
		}

		return nil
	},
}

// accountAddCmd represents the account add command
var accountAddCmd = &cobra.Command{
	Use:   "add [account-name]",
	Short: "Add a new account configuration",
	Long:  `Add a new Namecheap account configuration with an interactive prompt.`,
	Args:  cobra.MaximumNArgs(1),
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

		fmt.Printf("Adding new account: %s\n", accountName)
		fmt.Println("================================")
		fmt.Println()

		// Interactive input
		account := &config.AccountConfig{}

		fmt.Print("Namecheap Username: ")
		fmt.Scanln(&account.Username)

		fmt.Print("API User: ")
		fmt.Scanln(&account.APIUser)

		fmt.Print("API Key: ")
		fmt.Scanln(&account.APIKey)

		fmt.Print("Client IP Address: ")
		fmt.Scanln(&account.ClientIP)

		var sandboxInput string
		fmt.Print("Use Sandbox Environment? (y/N): ")
		fmt.Scanln(&sandboxInput)
		account.UseSandbox = strings.ToLower(sandboxInput) == "y" || strings.ToLower(sandboxInput) == "yes"

		fmt.Print("Description (optional): ")
		fmt.Scanln(&account.Description)

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
		fmt.Scanln(&switchInput)
		if switchInput == "" || strings.ToLower(switchInput) == "y" || strings.ToLower(switchInput) == "yes" {
			if err := configManager.SetCurrentAccount(accountName); err != nil {
				return fmt.Errorf("failed to switch to account '%s': %w", accountName, err)
			}
			fmt.Printf("✅ Switched to account '%s'\n", accountName)
		}

		return nil
	},
}

// accountSwitchCmd represents the account switch command
var accountSwitchCmd = &cobra.Command{
	Use:   "switch [account-name]",
	Short: "Switch to a different account",
	Long:  `Switch to a different configured Namecheap account.`,
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

		// Get current account for comparison
		currentAccount, err := configManager.GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get current account: %w", err)
		}

		if accountName == configManager.GetCurrentAccountName() {
			fmt.Printf("Already using account '%s'\n", accountName)
			return nil
		}

		// Switch account
		if err := configManager.SetCurrentAccount(accountName); err != nil {
			return fmt.Errorf("failed to switch to account '%s': %w", accountName, err)
		}

		fmt.Printf("✅ Switched from account '%s' to '%s'\n", currentAccount.Username, accountName)
		return nil
	},
}

// accountRemoveCmd represents the account remove command
var accountRemoveCmd = &cobra.Command{
	Use:   "remove [account-name]",
	Short: "Remove an account configuration",
	Long:  `Remove a Namecheap account configuration. Cannot remove the last remaining account.`,
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

		// Get current account for comparison
		currentAccount, err := configManager.GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get current account: %w", err)
		}

		// Confirm removal
		fmt.Printf("Are you sure you want to remove account '%s'? (y/N): ", accountName)
		var confirm string
		fmt.Scanln(&confirm)
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
		if accountName == currentAccount.Username {
			newCurrent, err := configManager.GetCurrentAccount()
			if err == nil {
				fmt.Printf("Switched to account '%s'\n", newCurrent.Username)
			}
		}

		return nil
	},
}

// accountShowCmd represents the account show command
var accountShowCmd = &cobra.Command{
	Use:   "show [account-name]",
	Short: "Show details of a specific account",
	Long:  `Display detailed information about a specific Namecheap account configuration.`,
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

		fmt.Printf("Username: %s\n", account.Username)
		fmt.Printf("API User: %s\n", account.APIUser)
		fmt.Printf("API Key: %s\n", config.MaskAPIKey(account.APIKey))
		fmt.Printf("Client IP: %s\n", account.ClientIP)
		fmt.Printf("Sandbox: %t\n", account.UseSandbox)
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
	Long:  `Edit an existing Namecheap account configuration with an interactive prompt.`,
	Args:  cobra.MaximumNArgs(1),
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

		fmt.Printf("Editing account: %s\n", accountName)
		fmt.Println("================================")
		fmt.Println()

		// Interactive input with current values as defaults
		account := &config.AccountConfig{}

		fmt.Printf("Namecheap Username [%s]: ", existingAccount.Username)
		var input string
		fmt.Scanln(&input)
		if input != "" {
			account.Username = input
		} else {
			account.Username = existingAccount.Username
		}

		fmt.Printf("API User [%s]: ", existingAccount.APIUser)
		fmt.Scanln(&input)
		if input != "" {
			account.APIUser = input
		} else {
			account.APIUser = existingAccount.APIUser
		}

		masked := existingAccount.APIKey
		if len(existingAccount.APIKey) > 4 {
			masked = existingAccount.APIKey[:4]
		}
		fmt.Printf("API Key [%s***]: ", masked)
		fmt.Scanln(&input)
		if input != "" {
			account.APIKey = input
		} else {
			account.APIKey = existingAccount.APIKey
		}

		fmt.Printf("Client IP Address [%s]: ", existingAccount.ClientIP)
		fmt.Scanln(&input)
		if input != "" {
			account.ClientIP = input
		} else {
			account.ClientIP = existingAccount.ClientIP
		}

		fmt.Printf("Use Sandbox Environment? [%t] (y/N): ", existingAccount.UseSandbox)
		fmt.Scanln(&input)
		if input != "" {
			account.UseSandbox = strings.ToLower(input) == "y" || strings.ToLower(input) == "yes"
		} else {
			account.UseSandbox = existingAccount.UseSandbox
		}

		fmt.Printf("Description [%s]: ", existingAccount.Description)
		fmt.Scanln(&input)
		if input != "" {
			account.Description = input
		} else {
			account.Description = existingAccount.Description
		}

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
}
