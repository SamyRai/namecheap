package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"zonekit/pkg/config"
	"zonekit/pkg/dns/provider/autodiscover"
	"zonekit/pkg/plugin"
	"zonekit/pkg/plugin/service"
	"zonekit/pkg/version"
)

var cfgFile string
var accountName string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zonekit",
	Short: "A CLI tool for managing DNS zones and records across multiple providers",
	Long: `A command-line interface for managing DNS zones and records across multiple providers.
This tool allows you to:
- List and manage your domains
- Create, update, and delete DNS records
- Bulk operations on DNS records
- Domain registration and management
- Manage multiple DNS provider accounts
- Support for multiple DNS providers (Namecheap, Cloudflare, and more)

Current version: ` + version.Version + ` (pre-1.0.0)`,
	Version: version.String(),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initProviders, initPlugins)

	// Here you will define your flags and configuration settings.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zonekit.yaml)")
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

			// Search config in home directory with name ".zonekit" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".zonekit")
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

// initProviders registers all available DNS providers
func initProviders() {
	// Auto-discover and register all REST-based providers from subdirectories
	// OpenAPI-only approach: providers must have openapi.yaml file
	if err := autodiscover.DiscoverAndRegister(""); err != nil {
		// Log but don't fail - some providers might not have OpenAPI specs
		// This is expected in development or if providers aren't configured
	}
}

// initPlugins registers all built-in plugins
func initPlugins() {
	// Register generic service plugin with config-based service integrations
	serviceConfigs, err := loadServiceConfigs()
	if err != nil {
		// Log error but don't fail - service plugin is optional
		fmt.Fprintf(os.Stderr, "Warning: failed to load service configs: %v\n", err)
	} else if len(serviceConfigs) > 0 {
		servicePlugin := service.NewServicePlugin(serviceConfigs)
		if err := plugin.Register(servicePlugin); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to register service plugin: %v\n", err)
		}
	}
}

// loadServiceConfigs loads all service integration configurations
func loadServiceConfigs() (map[string]*service.Config, error) {
	// Try to find services directory relative to executable or project root
	servicesDir := findServicesDirectory()
	if servicesDir == "" {
		return nil, fmt.Errorf("services directory not found")
	}

	configs, err := service.LoadAllConfigs(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load service configs: %w", err)
	}

	return configs, nil
}

// findServicesDirectory finds the services configuration directory
func findServicesDirectory() string {
	// Try multiple locations in order of preference:

	// 1. Project directory (for development)
	if projectConfigPath := config.FindProjectConfigPath(); projectConfigPath != "" {
		projectRoot := filepath.Dir(filepath.Dir(projectConfigPath))
		servicesPath := filepath.Join(projectRoot, "pkg", "plugin", "service", "services")
		if _, err := os.Stat(servicesPath); err == nil {
			return servicesPath
		}
	}

	// 2. Relative to executable (for installed binaries)
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		servicesPath := filepath.Join(exeDir, "services")
		if _, err := os.Stat(servicesPath); err == nil {
			return servicesPath
		}
		// Also try embedded location
		servicesPath = filepath.Join(exeDir, "pkg", "plugin", "service", "services")
		if _, err := os.Stat(servicesPath); err == nil {
			return servicesPath
		}
	}

	// 3. Current working directory
	cwd, _ := os.Getwd()
	servicesPath := filepath.Join(cwd, "pkg", "plugin", "service", "services")
	if _, err := os.Stat(servicesPath); err == nil {
		return servicesPath
	}

	// 4. Home directory
	if home, err := os.UserHomeDir(); err == nil {
		servicesPath := filepath.Join(home, ".zonekit", "services")
		if _, err := os.Stat(servicesPath); err == nil {
			return servicesPath
		}
	}

	return ""
}
