package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// helpCmd represents the help command
var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show help and examples",
	Long:  `Show help information including multi-account usage examples.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("ZoneKit - Multi-Provider DNS Management CLI")
		fmt.Println("==========================================")
		fmt.Println()

		fmt.Println("🎯 Key Features:")
		fmt.Println("• Support for multiple DNS providers (Namecheap, Cloudflare, and more)")
		fmt.Println("• Manage multiple provider accounts")
		fmt.Println("• Easy account switching")
		fmt.Println("• Domain and DNS management")
		fmt.Println("• Secure configuration storage")
		fmt.Println()

		fmt.Println("📋 Account Management Commands:")
		fmt.Println("  zonekit account list                    - List all configured accounts")
		fmt.Println("  zonekit account add [name]              - Add a new account")
		fmt.Println("  zonekit account switch <name>           - Switch to a different account")
		fmt.Println("  zonekit account show [name]             - Show account details")
		fmt.Println("  zonekit account edit [name]             - Edit account configuration")
		fmt.Println("  zonekit account remove <name>           - Remove an account")
		fmt.Println()

		fmt.Println("🌐 Domain Management Commands:")
		fmt.Println("  zonekit domain list                     - List all domains")
		fmt.Println("  zonekit domain info <domain>            - Get domain details")
		fmt.Println("  zonekit domain check <domain>           - Check domain availability")
		fmt.Println("  zonekit domain renew <domain> [years]   - Renew a domain")
		fmt.Println("  zonekit domain nameservers get <domain> - Get nameservers")
		fmt.Println("  zonekit domain nameservers set <domain> <ns1> [ns2] [ns3] [ns4]")
		fmt.Println("  zonekit domain nameservers default <domain>")
		fmt.Println()

		fmt.Println("🔧 DNS Management Commands:")
		fmt.Println("  zonekit dns list <domain>               - List DNS records")
		fmt.Println("  zonekit dns add <domain> <host> <type> <value>")
		fmt.Println("  zonekit dns update <domain> <host> <type> <value>")
		fmt.Println("  zonekit dns delete <domain> <host> <type>")
		fmt.Println("  zonekit dns clear <domain>              - Clear all records")
		fmt.Println("  zonekit dns bulk <domain> <file>        - Bulk operations")
		fmt.Println("  zonekit dns import <domain> <file>      - Import zone file")
		fmt.Println("  zonekit dns export <domain> [file]      - Export zone file")
		fmt.Println()

		fmt.Println("⚙️  Configuration Commands:")
		fmt.Println("  zonekit config init                      - Initialize config file")
		fmt.Println("  zonekit config set                       - Set configuration (legacy)")
		fmt.Println("  zonekit config show                      - Show configuration (legacy)")
		fmt.Println("  zonekit config validate                  - Validate configuration")
		fmt.Println()

		fmt.Println("🚀 Quick Start Examples:")
		fmt.Println()

		fmt.Println("1. First-time setup:")
		fmt.Println("   zonekit config init")
		fmt.Println("   # Edit ~/.zonekit.yaml with your credentials")
		fmt.Println("   zonekit account list")
		fmt.Println()

		fmt.Println("2. Add multiple accounts:")
		fmt.Println("   zonekit account add personal")
		fmt.Println("   zonekit account add work")
		fmt.Println("   zonekit account add client1")
		fmt.Println()

		fmt.Println("3. Switch between accounts:")
		fmt.Println("   zonekit account switch work")
		fmt.Println("   zonekit domain list")
		fmt.Println("   zonekit account switch personal")
		fmt.Println("   zonekit domain list")
		fmt.Println()

		fmt.Println("4. Use specific account for a command:")
		fmt.Println("   zonekit --account work domain list")
		fmt.Println("   zonekit --account personal dns list example.com")
		fmt.Println()

		fmt.Println("5. Manage DNS records:")
		fmt.Println("   zonekit dns add example.com www A 192.168.1.1")
		fmt.Println("   zonekit dns add example.com mail MX 192.168.1.2 --mx-pref 10")
		fmt.Println("   zonekit dns list example.com")
		fmt.Println()

		fmt.Println("🔐 Security Features:")
		fmt.Println("• API keys are masked in output")
		fmt.Println("• Configuration files use 600 permissions")
		fmt.Println("• Account credentials are encrypted in memory")
		fmt.Println()

		fmt.Println("📁 Configuration File:")
		fmt.Println("• Location: ~/.zonekit.yaml")
		fmt.Println("• Format: YAML with multi-account support")
		fmt.Println("• Automatic migration from legacy format")
		fmt.Println()

		fmt.Println("💡 Pro Tips:")
		fmt.Println("• Use descriptive account names (e.g., 'personal', 'work', 'client1')")
		fmt.Println("• Add descriptions to accounts for better organization")
		fmt.Println("• Use the --account flag for one-off commands with different accounts")
		fmt.Println("• Check 'zonekit account list' to see all available accounts")
		fmt.Println("• Use 'zonekit account show' to verify current account details")
		fmt.Println()

		fmt.Println("🆘 Need Help?")
		fmt.Println("• Run 'zonekit --help' for command overview")
		fmt.Println("• Run 'zonekit <command> --help' for specific command help")
		fmt.Println("• Check the README.md file for detailed documentation")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(helpCmd)
}
