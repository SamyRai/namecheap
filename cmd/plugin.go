package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"zonekit/internal/cmdutil"
	"zonekit/pkg/dns"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/plugin"
)

// pluginCmd represents the plugin command
var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage and use plugins",
	Long:  `Commands for managing and using plugins that extend functionality.`,
}

// pluginListCmd lists all available plugins
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  `Display all registered plugins and their commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		plugins := plugin.List()

		if len(plugins) == 0 {
			fmt.Println("No plugins registered.")
			return nil
		}

		fmt.Println("Available Plugins:")
		fmt.Println("==================")
		fmt.Println()

		for _, p := range plugins {
			fmt.Printf("📦 %s (v%s)\n", p.Name(), p.Version())
			fmt.Printf("   %s\n", p.Description())
			fmt.Println()

			commands := p.Commands()
			if len(commands) > 0 {
				fmt.Println("   Commands:")
				for _, cmd := range commands {
					fmt.Printf("     • %s - %s\n", cmd.Name, cmd.Description)
				}
				fmt.Println()
			}
		}

		return nil
	},
}

// pluginInfoCmd shows information about a specific plugin
var pluginInfoCmd = &cobra.Command{
	Use:   "info <plugin-name>",
	Short: "Show information about a plugin",
	Long:  `Display detailed information about a specific plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		p, err := plugin.Get(pluginName)
		if err != nil {
			return fmt.Errorf("plugin not found: %w", err)
		}

		fmt.Printf("Plugin: %s\n", p.Name())
		fmt.Printf("Version: %s\n", p.Version())
		fmt.Printf("Description: %s\n", p.Description())
		fmt.Println()

		commands := p.Commands()
		if len(commands) > 0 {
			fmt.Println("Commands:")
			fmt.Println("=========")
			for _, cmd := range commands {
				fmt.Printf("\n%s\n", cmd.Name)
				fmt.Printf("  %s\n", cmd.Description)
				if cmd.LongDescription != "" {
					fmt.Printf("  %s\n", cmd.LongDescription)
				}
			}
		}

		return nil
	},
}

// pluginExecuteCmd executes a plugin command
var pluginExecuteCmd = &cobra.Command{
	Use:   "<plugin-name> <command> <domain> [args...]",
	Short: "Execute a plugin command",
	Long:  `Execute a command from a specific plugin.`,
	Args:  cobra.MinimumNArgs(3), // plugin-name, command, domain
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		commandName := args[1]
		domainName := args[2]
		extraArgs := args[3:]

		// Get plugin
		p, err := plugin.Get(pluginName)
		if err != nil {
			return fmt.Errorf("plugin not found: %w", err)
		}

		// Find command
		var pluginCmd plugin.Command
		found := false
		for _, c := range p.Commands() {
			if c.Name == commandName {
				pluginCmd = c
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("command '%s' not found in plugin '%s'", commandName, pluginName)
		}

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

		// Build flags map from cobra command flags
		flags := make(map[string]interface{})

		// Get common flags
		if cmd.Flags().Changed("dry-run") {
			val, _ := cmd.Flags().GetBool("dry-run")
			flags["dry-run"] = val
		}
		if cmd.Flags().Changed("replace") {
			val, _ := cmd.Flags().GetBool("replace")
			flags["replace"] = val
		}
		if cmd.Flags().Changed("confirm") {
			val, _ := cmd.Flags().GetBool("confirm")
			flags["confirm"] = val
		}

		// Create context - wrap DNS service to match interface
		ctx := &plugin.Context{
			Domain: domainName,
			DNS:    &dnsServiceWrapper{service: dnsService},
			Args:   extraArgs,
			Flags:  flags,
			Output: &outputWriter{},
		}

		// Execute command
		return pluginCmd.Execute(ctx)
	},
}

// outputWriter implements plugin.OutputWriter
type outputWriter struct{}

func (w *outputWriter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (w *outputWriter) Println(args ...interface{}) {
	fmt.Println(args...)
}

func (w *outputWriter) Print(args ...interface{}) {
	fmt.Print(args...)
}

// dnsServiceWrapper wraps dns.Service to match plugin.Service interface
type dnsServiceWrapper struct {
	service *dns.Service
}

func (w *dnsServiceWrapper) GetRecords(domainName string) ([]dnsrecord.Record, error) {
	return w.service.GetRecords(context.Background(), domainName)
}

func (w *dnsServiceWrapper) GetRecordsByType(domainName string, recordType string) ([]dnsrecord.Record, error) {
	return w.service.GetRecordsByType(context.Background(), domainName, recordType)
}

func (w *dnsServiceWrapper) SetRecords(domainName string, records []dnsrecord.Record) error {
	return w.service.SetRecords(context.Background(), domainName, records)
}

func (w *dnsServiceWrapper) AddRecord(domainName string, record dnsrecord.Record) error {
	return w.service.AddRecord(context.Background(), domainName, record)
}

func (w *dnsServiceWrapper) UpdateRecord(domainName string, hostname, recordType string, newRecord dnsrecord.Record) error {
	return w.service.UpdateRecord(context.Background(), domainName, hostname, recordType, newRecord)
}

func (w *dnsServiceWrapper) DeleteRecord(domainName string, hostname, recordType string) error {
	return w.service.DeleteRecord(context.Background(), domainName, hostname, recordType)
}

func (w *dnsServiceWrapper) DeleteAllRecords(domainName string) error {
	return w.service.DeleteAllRecords(context.Background(), domainName)
}

func (w *dnsServiceWrapper) ValidateRecord(record dnsrecord.Record) error {
	return w.service.ValidateRecord(record)
}

func (w *dnsServiceWrapper) BulkUpdate(domainName string, operations []dns.BulkOperation) error {
	return w.service.BulkUpdate(context.Background(), domainName, operations)
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	pluginCmd.AddCommand(pluginExecuteCmd)

	// Add common flags for plugin commands
	pluginExecuteCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	pluginExecuteCmd.Flags().Bool("replace", false, "Replace existing records")
	pluginExecuteCmd.Flags().BoolP("confirm", "y", false, "Confirm the operation")
}
