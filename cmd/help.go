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
		fmt.Println("Namecheap DNS Manager - Multi-Account CLI Tool")
		fmt.Println("==============================================")
		fmt.Println()

		fmt.Println("🎯 Key Features:")
		fmt.Println("• Manage multiple Namecheap accounts")
		fmt.Println("• Easy account switching")
		fmt.Println("• Domain and DNS management")
		fmt.Println("• Secure configuration storage")
		fmt.Println()

		fmt.Println("📋 Account Management Commands:")
		fmt.Println("  namecheap-dns account list                    - List all configured accounts")
		fmt.Println("  namecheap-dns account add [name]              - Add a new account")
		fmt.Println("  namecheap-dns account switch <name>           - Switch to a different account")
		fmt.Println("  namecheap-dns account show [name]             - Show account details")
		fmt.Println("  namecheap-dns account edit [name]             - Edit account configuration")
		fmt.Println("  namecheap-dns account remove <name>           - Remove an account")
		fmt.Println()

		fmt.Println("🌐 Domain Management Commands:")
		fmt.Println("  namecheap-dns domain list                     - List all domains")
		fmt.Println("  namecheap-dns domain info <domain>            - Get domain details")
		fmt.Println("  namecheap-dns domain check <domain>           - Check domain availability")
		fmt.Println("  namecheap-dns domain renew <domain> [years]   - Renew a domain")
		fmt.Println("  namecheap-dns domain nameservers get <domain> - Get nameservers")
		fmt.Println("  namecheap-dns domain nameservers set <domain> <ns1> [ns2] [ns3] [ns4]")
		fmt.Println("  namecheap-dns domain nameservers default <domain>")
		fmt.Println()

		fmt.Println("🔧 DNS Management Commands:")
		fmt.Println("  namecheap-dns dns list <domain>               - List DNS records")
		fmt.Println("  namecheap-dns dns add <domain> <host> <type> <value>")
		fmt.Println("  namecheap-dns dns update <domain> <host> <type> <value>")
		fmt.Println("  namecheap-dns dns delete <domain> <host> <type>")
		fmt.Println("  namecheap-dns dns clear <domain>              - Clear all records")
		fmt.Println("  namecheap-dns dns bulk <domain> <file>        - Bulk operations")
		fmt.Println("  namecheap-dns dns import <domain> <file>      - Import zone file")
		fmt.Println("  namecheap-dns dns export <domain> [file]      - Export zone file")
		fmt.Println()

		fmt.Println("⚙️  Configuration Commands:")
		fmt.Println("  namecheap-dns config init                      - Initialize config file")
		fmt.Println("  namecheap-dns config set                       - Set configuration (legacy)")
		fmt.Println("  namecheap-dns config show                      - Show configuration (legacy)")
		fmt.Println("  namecheap-dns config validate                  - Validate configuration")
		fmt.Println()

		fmt.Println("🚀 Quick Start Examples:")
		fmt.Println()

		fmt.Println("1. First-time setup:")
		fmt.Println("   namecheap-dns config init")
		fmt.Println("   # Edit ~/.namecheap-dns.yaml with your credentials")
		fmt.Println("   namecheap-dns account list")
		fmt.Println()

		fmt.Println("2. Add multiple accounts:")
		fmt.Println("   namecheap-dns account add personal")
		fmt.Println("   namecheap-dns account add work")
		fmt.Println("   namecheap-dns account add client1")
		fmt.Println()

		fmt.Println("3. Switch between accounts:")
		fmt.Println("   namecheap-dns account switch work")
		fmt.Println("   namecheap-dns domain list")
		fmt.Println("   namecheap-dns account switch personal")
		fmt.Println("   namecheap-dns domain list")
		fmt.Println()

		fmt.Println("4. Use specific account for a command:")
		fmt.Println("   namecheap-dns --account work domain list")
		fmt.Println("   namecheap-dns --account personal dns list example.com")
		fmt.Println()

		fmt.Println("5. Manage DNS records:")
		fmt.Println("   namecheap-dns dns add example.com www A 192.168.1.1")
		fmt.Println("   namecheap-dns dns add example.com mail MX 192.168.1.2 --mx-pref 10")
		fmt.Println("   namecheap-dns dns list example.com")
		fmt.Println()

		fmt.Println("🔐 Security Features:")
		fmt.Println("• API keys are masked in output")
		fmt.Println("• Configuration files use 600 permissions")
		fmt.Println("• Account credentials are encrypted in memory")
		fmt.Println()

		fmt.Println("📁 Configuration File:")
		fmt.Println("• Location: ~/.namecheap-dns.yaml")
		fmt.Println("• Format: YAML with multi-account support")
		fmt.Println("• Automatic migration from legacy format")
		fmt.Println()

		fmt.Println("💡 Pro Tips:")
		fmt.Println("• Use descriptive account names (e.g., 'personal', 'work', 'client1')")
		fmt.Println("• Add descriptions to accounts for better organization")
		fmt.Println("• Use the --account flag for one-off commands with different accounts")
		fmt.Println("• Check 'namecheap-dns account list' to see all available accounts")
		fmt.Println("• Use 'namecheap-dns account show' to verify current account details")
		fmt.Println()

		fmt.Println("🆘 Need Help?")
		fmt.Println("• Run 'namecheap-dns --help' for command overview")
		fmt.Println("• Run 'namecheap-dns <command> --help' for specific command help")
		fmt.Println("• Check the README.md file for detailed documentation")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(helpCmd)
}
