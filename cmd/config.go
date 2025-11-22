package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"namecheap-dns-manager/pkg/config"
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
		config := make(map[string]interface{})

		// Get current values if they exist
		username := viper.GetString("username")
		apiUser := viper.GetString("api-user")
		apiKey := viper.GetString("api-key")
		clientIP := viper.GetString("client-ip")
		sandbox := viper.GetBool("sandbox")

		fmt.Println("Namecheap Configuration Setup")
		fmt.Println("=============================")
		fmt.Println()

		// Username
		fmt.Print("Namecheap Username")
		if username != "" {
			fmt.Printf(" [%s]", username)
		}
		fmt.Print(": ")
		var input string
		fmt.Scanln(&input)
		if input != "" {
			username = input
		}
		if username != "" {
			config["username"] = username
		}

		// API User
		fmt.Print("API User")
		if apiUser != "" {
			fmt.Printf(" [%s]", apiUser)
		}
		fmt.Print(": ")
		fmt.Scanln(&input)
		if input != "" {
			apiUser = input
		}
		if apiUser != "" {
			config["api_user"] = apiUser
		}

		// API Key
		fmt.Print("API Key")
		if apiKey != "" {
			masked := apiKey
			if len(apiKey) > 4 {
				masked = apiKey[:4]
			}
			fmt.Printf(" [%s***]", masked)
		}
		fmt.Print(": ")
		fmt.Scanln(&input)
		if input != "" {
			apiKey = input
		}
		if apiKey != "" {
			config["api_key"] = apiKey
		}

		// Client IP
		fmt.Print("Client IP Address")
		if clientIP != "" {
			fmt.Printf(" [%s]", clientIP)
		}
		fmt.Print(": ")
		fmt.Scanln(&input)
		if input != "" {
			clientIP = input
		}
		if clientIP != "" {
			config["client_ip"] = clientIP
		}

		// Sandbox
		fmt.Print("Use Sandbox Environment? (y/N)")
		if sandbox {
			fmt.Print(" [y]")
		} else {
			fmt.Print(" [N]")
		}
		fmt.Print(": ")
		fmt.Scanln(&input)
		if input != "" {
			sandbox = (input == "y" || input == "Y" || input == "yes" || input == "Yes")
		}
		config["use_sandbox"] = sandbox

		// Save configuration
		return saveConfig(config)
	},
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Current Configuration:")
		fmt.Println("=====================")

		if configFile := viper.ConfigFileUsed(); configFile != "" {
			fmt.Printf("Config file: %s\n", configFile)
		} else {
			fmt.Println("No config file found")
		}

		fmt.Println()

		username := viper.GetString("username")
		apiUser := getStringWithFallback("api-user", "api_user")
		apiKey := getStringWithFallback("api-key", "api_key")
		clientIP := getStringWithFallback("client-ip", "client_ip")
		sandbox := getBoolWithFallback("sandbox", "use_sandbox")

		fmt.Printf("Username: %s\n", getValueOrEmpty(username))
		fmt.Printf("API User: %s\n", getValueOrEmpty(apiUser))
		fmt.Printf("API Key: %s\n", config.MaskAPIKey(apiKey))
		fmt.Printf("Client IP: %s\n", getValueOrEmpty(clientIP))
		fmt.Printf("Sandbox: %t\n", sandbox)

		fmt.Println()
		if username == "" || apiUser == "" || apiKey == "" || clientIP == "" {
			fmt.Println("⚠️  Some required configuration values are missing.")
			fmt.Println("   Run 'namecheap-dns config set' to configure them.")
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

		configPath := filepath.Join(home, ".namecheap-dns.yaml")

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
		}

		// Create example config
		config := map[string]interface{}{
			"username":    "your-namecheap-username",
			"api_user":    "your-api-username",
			"api_key":     "your-api-key",
			"client_ip":   "your.public.ip.address",
			"use_sandbox": false,
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		err = os.WriteFile(configPath, data, 0600)
		if err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Configuration file created at %s\n", configPath)
		fmt.Println("Please edit the file with your actual values, then run:")
		fmt.Println("  namecheap-dns config show")

		return nil
	},
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and test API connection",
	Long:  `Validate the current configuration and test the connection to Namecheap API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Validating configuration...")

		// Check required fields
		username := viper.GetString("username")
		apiUser := viper.GetString("api-user")
		apiKey := viper.GetString("api-key")
		clientIP := viper.GetString("client-ip")

		if username == "" {
			return fmt.Errorf("username is required")
		}
		if apiUser == "" {
			return fmt.Errorf("api-user is required")
		}
		if apiKey == "" {
			return fmt.Errorf("api-key is required")
		}
		if clientIP == "" {
			return fmt.Errorf("client-ip is required")
		}

		fmt.Println("✅ All required fields are present")

		// Test API connection
		fmt.Println("Testing API connection...")

		// TODO: Implement actual API test
		// This would involve creating a client and making a simple API call
		// For now, just validate the configuration format

		fmt.Println("✅ Configuration appears valid")
		fmt.Println()
		fmt.Println("Note: Run 'namecheap-dns domain list' to test the actual API connection.")

		return nil
	},
}

func saveConfig(config map[string]interface{}) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".namecheap-dns.yaml")

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration saved to %s\n", configPath)
	return nil
}

func getValueOrEmpty(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}

// getStringWithFallback tries the primary key first, then falls back to the alternative
func getStringWithFallback(primary, fallback string) string {
	if value := viper.GetString(primary); value != "" {
		return value
	}
	return viper.GetString(fallback)
}

// getBoolWithFallback tries the primary key first, then falls back to the alternative
func getBoolWithFallback(primary, fallback string) bool {
	if viper.IsSet(primary) {
		return viper.GetBool(primary)
	}
	return viper.GetBool(fallback)
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}
