package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"zonekit/internal/cmdutil"
	"zonekit/pkg/dns"
	"zonekit/pkg/plugin"
)

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Service integration setup and management",
	Long:  `Commands for setting up DNS records for various service integrations (email, CDN, hosting, etc.) using config-based templates.`,
}

// serviceListCmd lists all available service integrations
var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available service integrations",
	Long:  `Display all configured service integrations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get service plugin
		p, err := plugin.Get("service")
		if err != nil {
			return fmt.Errorf("service plugin not found: %w", err)
		}

		// Create context for list command
		ctx := &plugin.Context{
			Domain: "",
			DNS:    nil,
			Args:   []string{},
			Flags:  make(map[string]interface{}),
			Output: &outputWriter{},
		}

		// Find and execute list command
		for _, pluginCmd := range p.Commands() {
			if pluginCmd.Name == "list" {
				return pluginCmd.Execute(ctx)
			}
		}

		return fmt.Errorf("list command not found in service plugin")
	},
}

// serviceInfoCmd shows information about a specific service integration
var serviceInfoCmd = &cobra.Command{
	Use:   "info <service-name>",
	Short: "Show service integration information",
	Long:  `Display detailed information about a specific service integration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]

		// Get service plugin
		p, err := plugin.Get("service")
		if err != nil {
			return fmt.Errorf("service plugin not found: %w", err)
		}

		// Create context for info command
		ctx := &plugin.Context{
			Domain: "",
			DNS:    nil,
			Args:   []string{serviceName},
			Flags:  make(map[string]interface{}),
			Output: &outputWriter{},
		}

		// Find and execute info command
		for _, pluginCmd := range p.Commands() {
			if pluginCmd.Name == "info" {
				return pluginCmd.Execute(ctx)
			}
		}

		return fmt.Errorf("info command not found in service plugin")
	},
}

// serviceSetupCmd sets up DNS records for a service integration
var serviceSetupCmd = &cobra.Command{
	Use:   "setup <service-name> <domain>",
	Short: "Set up DNS records for a service integration",
	Long:  `Set up all necessary DNS records for a configured service integration.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		domainName := args[1]

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		// Create DNS service
		dnsService := dns.NewService(client)

		// Build flags map
		flags := make(map[string]interface{})
		if cmd.Flags().Changed("dry-run") {
			val, _ := cmd.Flags().GetBool("dry-run")
			flags["dry-run"] = val
		}
		if cmd.Flags().Changed("replace") {
			val, _ := cmd.Flags().GetBool("replace")
			flags["replace"] = val
		}

		// Get service plugin
		p, err := plugin.Get("service")
		if err != nil {
			return fmt.Errorf("service plugin not found: %w", err)
		}

		// Create context
		ctx := &plugin.Context{
			Domain: domainName,
			DNS:    &dnsServiceWrapper{service: dnsService},
			Args:   []string{serviceName, domainName},
			Flags:  flags,
			Output: &outputWriter{},
		}

		// Find and execute setup command
		for _, pluginCmd := range p.Commands() {
			if pluginCmd.Name == "setup" {
				return pluginCmd.Execute(ctx)
			}
		}

		return fmt.Errorf("setup command not found in service plugin")
	},
}

// serviceVerifyCmd verifies DNS records for a service integration
var serviceVerifyCmd = &cobra.Command{
	Use:   "verify <service-name> <domain>",
	Short: "Verify DNS records for a service integration",
	Long:  `Check if all required DNS records for a service integration are properly configured.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		domainName := args[1]

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		// Create DNS service
		dnsService := dns.NewService(client)

		// Get service plugin
		p, err := plugin.Get("service")
		if err != nil {
			return fmt.Errorf("service plugin not found: %w", err)
		}

		// Create context
		ctx := &plugin.Context{
			Domain: domainName,
			DNS:    &dnsServiceWrapper{service: dnsService},
			Args:   []string{serviceName, domainName},
			Flags:  make(map[string]interface{}),
			Output: &outputWriter{},
		}

		// Find and execute verify command
		for _, pluginCmd := range p.Commands() {
			if pluginCmd.Name == "verify" {
				return pluginCmd.Execute(ctx)
			}
		}

		return fmt.Errorf("verify command not found in service plugin")
	},
}

// serviceRemoveCmd removes DNS records for a service integration
var serviceRemoveCmd = &cobra.Command{
	Use:   "remove <service-name> <domain>",
	Short: "Remove DNS records for a service integration",
	Long:  `Remove all service-related DNS records from the specified domain.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		domainName := args[1]

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		// Create DNS service
		dnsService := dns.NewService(client)

		// Build flags map
		flags := make(map[string]interface{})
		if cmd.Flags().Changed("confirm") {
			val, _ := cmd.Flags().GetBool("confirm")
			flags["confirm"] = val
		}

		// Get service plugin
		p, err := plugin.Get("service")
		if err != nil {
			return fmt.Errorf("service plugin not found: %w", err)
		}

		// Create context
		ctx := &plugin.Context{
			Domain: domainName,
			DNS:    &dnsServiceWrapper{service: dnsService},
			Args:   []string{serviceName, domainName},
			Flags:  flags,
			Output: &outputWriter{},
		}

		// Find and execute remove command
		for _, pluginCmd := range p.Commands() {
			if pluginCmd.Name == "remove" {
				return pluginCmd.Execute(ctx)
			}
		}

		return fmt.Errorf("remove command not found in service plugin")
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceInfoCmd)
	serviceCmd.AddCommand(serviceSetupCmd)
	serviceCmd.AddCommand(serviceVerifyCmd)
	serviceCmd.AddCommand(serviceRemoveCmd)

	// Flags
	serviceSetupCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	serviceSetupCmd.Flags().Bool("replace", false, "Replace existing records")
	serviceRemoveCmd.Flags().BoolP("confirm", "y", false, "Confirm the operation")
}
