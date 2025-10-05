package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"namecheap-dns-manager/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var accountName string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "namecheap-dns",
	Short: "A CLI tool for managing Namecheap domains and DNS records",
	Long: `A comprehensive command-line interface for managing Namecheap domains and DNS records.
This tool allows you to:
- List and manage your domains
- Create, update, and delete DNS records
- Bulk operations on DNS records
- Domain registration and management
- Manage multiple Namecheap accounts`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.namecheap-dns.yaml)")
	rootCmd.PersistentFlags().StringVar(&accountName, "account", "", "use specific account (default: current account)")

	// Legacy flags for backward compatibility (deprecated)
	rootCmd.PersistentFlags().String("username", "", "Namecheap username (deprecated: use account management)")
	rootCmd.PersistentFlags().String("api-user", "", "Namecheap API user (deprecated: use account management)")
	rootCmd.PersistentFlags().String("api-key", "", "Namecheap API key (deprecated: use account management)")
	rootCmd.PersistentFlags().String("client-ip", "", "Client IP address (deprecated: use account management)")
	rootCmd.PersistentFlags().Bool("sandbox", false, "Use Namecheap sandbox environment (deprecated: use account management)")

	// Mark legacy flags as deprecated
	rootCmd.PersistentFlags().MarkDeprecated("username", "use account management instead")
	rootCmd.PersistentFlags().MarkDeprecated("api-user", "use account management instead")
	rootCmd.PersistentFlags().MarkDeprecated("api-key", "use account management instead")
	rootCmd.PersistentFlags().MarkDeprecated("client-ip", "use account management instead")
	rootCmd.PersistentFlags().MarkDeprecated("sandbox", "use account management instead")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// First try to find config in project directory
		projectConfigPath := findProjectConfigPath()
		if projectConfigPath != "" {
			viper.SetConfigFile(projectConfigPath)
		} else {
			// Find home directory.
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)

			// Search config in home directory with name ".namecheap-dns" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".namecheap-dns")
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// findProjectConfigPath looks for config file in the project directory
func findProjectConfigPath() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Look for config in current directory and parent directories
	for {
		configPath := filepath.Join(cwd, "configs", ".namecheap-dns.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// Move up one directory
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break // Reached root
		}
		cwd = parent
	}

	return ""
}

// GetConfigManager returns a configuration manager instance
func GetConfigManager() (*config.Manager, error) {
	return config.NewManager()
}

// GetCurrentAccount returns the current account configuration
func GetCurrentAccount() (*config.AccountConfig, error) {
	configManager, err := GetConfigManager()
	if err != nil {
		return nil, err
	}

	// If account flag is specified, use that account
	if accountName != "" {
		return configManager.GetAccount(accountName)
	}

	// Otherwise use the current account
	return configManager.GetCurrentAccount()
}
