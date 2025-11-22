package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"namecheap-dns-manager/pkg/config"
	"namecheap-dns-manager/pkg/plugin"
	"namecheap-dns-manager/pkg/plugin/migadu"
	"namecheap-dns-manager/pkg/version"
)

var cfgFile string
var accountName string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "namecheap-dns",
	Short: "A CLI tool for managing Namecheap domains and DNS records",
	Long: `A command-line interface for managing Namecheap domains and DNS records.
This tool allows you to:
- List and manage your domains
- Create, update, and delete DNS records
- Bulk operations on DNS records
- Domain registration and management
- Manage multiple Namecheap accounts

⚠️  WARNING: This is NOT an official Namecheap tool. Use at your own risk.
Current version: ` + version.Version + ` (pre-1.0.0)`,
	Version: version.String(),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initPlugins)

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
		projectConfigPath := config.FindProjectConfigPath()
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

// GetConfigManager returns a configuration manager instance
func GetConfigManager() (*config.Manager, error) {
	// If config file is specified via flag, use it
	if cfgFile != "" {
		return config.NewManagerWithPath(cfgFile)
	}
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

// initPlugins registers all built-in plugins
func initPlugins() {
	// Register Migadu plugin
	if err := plugin.Register(migadu.New()); err != nil {
		// Log error but don't fail - plugins are optional
		fmt.Fprintf(os.Stderr, "Warning: failed to register Migadu plugin: %v\n", err)
	}
	// Add more plugin registrations here as needed
}
