package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"zonekit/internal/cmdutil"
	"zonekit/pkg/config"
	"zonekit/pkg/domain"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for managing configuration settings.`,
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values interactively",
	Long:  `Set configuration values through an interactive prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return err
		}
		accountName := configManager.GetCurrentAccountName()
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			accountConfig = &config.AccountConfig{Provider: "namecheap"}
			configManager.AddAccount(accountName, accountConfig)
		}

		fmt.Println("DNS Provider Configuration Setup")
		fmt.Printf("================================= (Account: %s)\n\n", accountName)

		// Provider
		fmt.Print("Provider")
		if accountConfig.Provider != "" {
			fmt.Printf(" [%s]", accountConfig.Provider)
		}
		fmt.Print(": ")
		var input string
		fmt.Scanln(&input)
		if input != "" {
			accountConfig.Provider = input
		}

		// Username
		fmt.Print("Provider Username")
		if accountConfig.Username != "" {
			fmt.Printf(" [%s]", accountConfig.Username)
		}
		fmt.Print(": ")
		input = ""
		fmt.Scanln(&input)
		if input != "" {
			accountConfig.Username = input
		}

		// API User
		fmt.Print("API User")
		if accountConfig.APIUser != "" {
			fmt.Printf(" [%s]", accountConfig.APIUser)
		}
		fmt.Print(": ")
		input = ""
		fmt.Scanln(&input)
		if input != "" {
			accountConfig.APIUser = input
		}

		// API Key
		fmt.Print("API Key")
		if accountConfig.APIKey != "" {
			masked := accountConfig.APIKey
			if len(accountConfig.APIKey) > 4 {
				masked = accountConfig.APIKey[:4]
			}
			fmt.Printf(" [%s***]", masked)
		}
		fmt.Print(": ")
		input = ""
		fmt.Scanln(&input)
		if input != "" {
			accountConfig.APIKey = input
		}

		// Client IP
		fmt.Print("Client IP Address")
		if accountConfig.ClientIP != "" {
			fmt.Printf(" [%s]", accountConfig.ClientIP)
		}
		fmt.Print(": ")
		input = ""
		fmt.Scanln(&input)
		if input != "" {
			accountConfig.ClientIP = input
		}

		// Sandbox
		fmt.Print("Use Sandbox Environment? (y/N)")
		if accountConfig.UseSandbox {
			fmt.Print(" [y]")
		} else {
			fmt.Print(" [N]")
		}
		fmt.Print(": ")
		input = ""
		fmt.Scanln(&input)
		if input != "" {
			accountConfig.UseSandbox = (input == "y" || input == "Y" || input == "yes" || input == "Yes")
		}

		// Save configuration
		err = configManager.SaveSecretsToKeyring(accountName, accountConfig)
		if err != nil {
			fmt.Printf("Warning: Failed to save secrets to keyring: %v\n", err)
			fmt.Println("Secrets will be stored in plain text in config file.")
		}
		
		err = configManager.UpdateAccount(accountName, accountConfig)
		if err != nil {
			return err
		}
		fmt.Printf("Configuration saved to %s\n", configManager.GetConfigPath())
		return nil
	},
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewManager()
		if err != nil {
			return err
		}
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			return err
		}

		fmt.Println("Current Configuration:")
		fmt.Println("=====================")
		fmt.Printf("Config file: %s\n", configManager.GetConfigPath())
		fmt.Printf("Account: %s\n\n", configManager.GetCurrentAccountName())

		fmt.Printf("Provider: %s\n", getValueOrEmpty(accountConfig.Provider))
		fmt.Printf("Username: %s\n", getValueOrEmpty(accountConfig.Username))
		fmt.Printf("API User: %s\n", getValueOrEmpty(accountConfig.APIUser))
		fmt.Printf("API Key: %s\n", config.MaskAPIKey(accountConfig.APIKey))
		fmt.Printf("Client IP: %s\n", getValueOrEmpty(accountConfig.ClientIP))
		fmt.Printf("Sandbox: %t\n", accountConfig.UseSandbox)

		fmt.Println()
		if accountConfig.Username == "" || accountConfig.APIUser == "" || accountConfig.APIKey == "" || accountConfig.ClientIP == "" {
			fmt.Println("⚠️  Some required configuration values are missing.")
			fmt.Println("   Run 'zonekit config set' to configure them.")
		} else {
			fmt.Println("✅ Configuration appears complete.")
		}

		return nil
	},
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new configuration file with example values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configPath := filepath.Join(home, ".zonekit.yaml")

		// Check if file already exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Configuration file already exists at %s\n", configPath)
			fmt.Print("Overwrite? (y/N): ")
			var input string
			fmt.Scanln(&input)
			if input != "y" && input != "Y" && input != "yes" && input != "Yes" {
				fmt.Println("Aborted.")
				return nil
			}
			// Delete existing to force fresh default
			os.Remove(configPath)
		}

		configManager, err := config.NewManagerWithPath(configPath)
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}

		err = configManager.Save()
		if err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Configuration file created at %s\n", configPath)
		fmt.Println("Please edit the file with your actual values, then run:")
		fmt.Println("  zonekit config show")

		return nil
	},
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and test API connection",
	Long:  `Validate the current configuration and test the connection to DNS provider API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Validating configuration...")

		configManager, err := config.NewManager()
		if err != nil {
			return err
		}
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			return err
		}

		if accountConfig.Username == "" {
			return fmt.Errorf("username is required")
		}
		if accountConfig.APIUser == "" {
			return fmt.Errorf("api-user is required")
		}
		if accountConfig.APIKey == "" {
			return fmt.Errorf("api-key is required")
		}
		if accountConfig.ClientIP == "" {
			return fmt.Errorf("client-ip is required")
		}

		fmt.Println("✅ All required fields are present")

		// Test API connection
		fmt.Println("Testing API connection...")

		testClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return fmt.Errorf("failed to create test client: %w", err)
		}

		// Test the connection by making a simple API call
		domainService, err := domain.NewService(testClient)
		if err != nil {
			return fmt.Errorf("failed to create test domain service: %w", err)
		}
		_, err = domainService.ListDomains(cmd.Context())
		if err != nil {
			return fmt.Errorf("API connection test failed: %w", err)
		}

		fmt.Printf("✅ API connection successful - Account: %s\n", testClient.GetAccountName())
		fmt.Println()
		fmt.Println("Note: Run 'zonekit domain list' to test the actual API connection.")

		return nil
	},
}

func getValueOrEmpty(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}
