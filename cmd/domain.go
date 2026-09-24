package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.glpx.pro/zonekit/internal/cmdutil"
	"go.glpx.pro/zonekit/pkg/domain"
)

// domainDTO is the stable, snake_case wire shape for a domain in json/yaml
// output (O1).
type domainDTO struct {
	Name       string `json:"name" yaml:"name"`
	User       string `json:"user,omitempty" yaml:"user,omitempty"`
	Created    string `json:"created,omitempty" yaml:"created,omitempty"`
	Expires    string `json:"expires,omitempty" yaml:"expires,omitempty"`
	IsExpired  bool   `json:"is_expired" yaml:"is_expired"`
	IsLocked   bool   `json:"is_locked" yaml:"is_locked"`
	AutoRenew  bool   `json:"auto_renew" yaml:"auto_renew"`
	WhoisGuard string `json:"whois_guard,omitempty" yaml:"whois_guard,omitempty"`
	IsPremium  bool   `json:"is_premium" yaml:"is_premium"`
	IsOurDNS   bool   `json:"is_our_dns" yaml:"is_our_dns"`
}

func newDomainDTO(d domain.Domain) domainDTO {
	return domainDTO{
		Name:       d.Name,
		User:       d.User,
		Created:    d.Created,
		Expires:    d.Expires,
		IsExpired:  d.IsExpired,
		IsLocked:   d.IsLocked,
		AutoRenew:  d.AutoRenew,
		WhoisGuard: d.WhoisGuard,
		IsPremium:  d.IsPremium,
		IsOurDNS:   d.IsOurDNS,
	}
}

func newDomainDTOs(domains []domain.Domain) []domainDTO {
	out := make([]domainDTO, len(domains))
	for i, d := range domains {
		out[i] = newDomainDTO(d)
	}
	return out
}

// domainCmd represents the domain command
var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage domains",
	Long:  `Commands for managing domains including listing, checking availability, and basic domain operations.`,
}

// domainListCmd represents the domain list command
var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all domains",
	Long:  `List all domains in your account with their details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format := OutputFormat()

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

		domainService := domain.NewService(client)
		domains, err := domainService.ListDomains()
		if err != nil {
			return fmt.Errorf("failed to list domains: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, newDomainDTOs(domains))
		}

		if len(domains) == 0 {
			fmt.Println("No domains found in your account.")
			return nil
		}

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tCREATED\tEXPIRES\tAUTO-RENEW\tLOCKED\tDNS")

		for _, d := range domains {
			autoRenew := "No"
			if d.AutoRenew {
				autoRenew = "Yes"
			}
			locked := "No"
			if d.IsLocked {
				locked = "Yes"
			}
			dns := "External"
			if d.IsOurDNS {
				dns = "Provider"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				d.Name, d.Created, d.Expires, autoRenew, locked, dns)
		}

		w.Flush()
		return nil
	},
}

// domainInfoCmd represents the domain info command
var domainInfoCmd = &cobra.Command{
	Use:   "info <domain>",
	Short: "Get detailed information about a domain",
	Long:  `Get detailed information about a specific domain.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		format := OutputFormat()

		// Validate domain
		if err := domain.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
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

		domainService := domain.NewService(client)
		domainInfo, err := domainService.GetDomainInfo(domainName)
		if err != nil {
			return fmt.Errorf("failed to get domain info: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, newDomainDTO(*domainInfo))
		}

		fmt.Printf("Domain: %s\n", domainInfo.Name)
		fmt.Printf("Owner: %s\n", domainInfo.User)
		fmt.Printf("Created: %s\n", domainInfo.Created)
		fmt.Printf("Expires: %s\n", domainInfo.Expires)
		fmt.Printf("Auto-Renew: %t\n", domainInfo.AutoRenew)
		fmt.Printf("Locked: %t\n", domainInfo.IsLocked)
		fmt.Printf("WhoisGuard: %s\n", domainInfo.WhoisGuard)
		fmt.Printf("Premium: %t\n", domainInfo.IsPremium)
		fmt.Printf("Using Provider DNS: %t\n", domainInfo.IsOurDNS)

		return nil
	},
}

// domainCheckCmd represents the domain check command
var domainCheckCmd = &cobra.Command{
	Use:   "check <domain>",
	Short: "Check domain availability",
	Long:  `Check if a domain is available for registration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]

		// Validate domain
		if err := domain.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
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

		domainService := domain.NewService(client)
		available, err := domainService.CheckAvailability(domainName)
		if err != nil {
			return fmt.Errorf("failed to check domain availability: %w", err)
		}

		if available {
			fmt.Printf("Domain '%s' is AVAILABLE for registration.\n", domainName)
		} else {
			fmt.Printf("Domain '%s' is NOT AVAILABLE.\n", domainName)
		}

		return nil
	},
}

// domainNameserversCmd represents the domain nameservers command
var domainNameserversCmd = &cobra.Command{
	Use:   "nameservers",
	Short: "Manage domain nameservers",
	Long:  `Commands for managing domain nameservers.`,
}

// domainNameserversGetCmd represents the domain nameservers get command
var domainNameserversGetCmd = &cobra.Command{
	Use:   "get <domain>",
	Short: "Get domain nameservers",
	Long:  `Get the current nameservers for a domain.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		format := OutputFormat()

		// Validate domain
		if err := domain.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
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

		domainService := domain.NewService(client)
		nameservers, err := domainService.GetNameservers(domainName)
		if err != nil {
			return fmt.Errorf("failed to get nameservers: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]interface{}{
				jsonKeyDomain: domainName,
				"nameservers": nameservers,
			})
		}

		fmt.Printf("Nameservers for %s:\n", domainName)
		for i, ns := range nameservers {
			fmt.Printf("%d. %s\n", i+1, ns)
		}

		return nil
	},
}

// domainNameserversSetCmd represents the domain nameservers set command
var domainNameserversSetCmd = &cobra.Command{
	Use:   "set <domain> <ns1> [ns2] [ns3] [ns4]",
	Short: "Set custom nameservers for a domain",
	Long:  `Set custom nameservers for a domain. You can specify 2-4 nameservers.`,
	Args:  cobra.RangeArgs(3, 5), // domain + 2-4 nameservers
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		nameservers := args[1:]
		format := OutputFormat()
		dryRun, _ := cmd.Flags().GetBool("dry-run")

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

		if dryRun {
			if format != cmdutil.OutputTable {
				return cmdutil.WriteResult(os.Stdout, format, map[string]interface{}{
					jsonKeyDomain: domainName,
					"nameservers": nameservers,
					jsonKeyStatus: "would-set",
				})
			}
			fmt.Printf("DRY RUN: would set nameservers for %s:\n", domainName)
			for i, ns := range nameservers {
				fmt.Printf("%d. %s\n", i+1, ns)
			}
			return nil
		}

		domainService := domain.NewService(client)
		err = domainService.SetNameservers(domainName, nameservers)
		if err != nil {
			return fmt.Errorf("failed to set nameservers: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]interface{}{
				jsonKeyDomain: domainName,
				"nameservers": nameservers,
				jsonKeyStatus: "set",
			})
		}

		fmt.Printf("Successfully set nameservers for %s:\n", domainName)
		for i, ns := range nameservers {
			fmt.Printf("%d. %s\n", i+1, ns)
		}

		return nil
	},
}

// domainNameserversDefaultCmd represents the domain nameservers default command
var domainNameserversDefaultCmd = &cobra.Command{
	Use:   "default <domain>",
	Short: "Set domain to use provider DNS",
	Long:  `Set the domain to use the provider's default DNS servers.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		format := OutputFormat()
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Validate domain
		if err := domain.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
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

		if dryRun {
			if format != cmdutil.OutputTable {
				return cmdutil.WriteResult(os.Stdout, format, map[string]string{
					jsonKeyDomain: domainName,
					jsonKeyStatus: "would-reset-to-provider-dns",
				})
			}
			fmt.Printf("DRY RUN: would set %s to use provider DNS servers.\n", domainName)
			return nil
		}

		domainService := domain.NewService(client)
		err = domainService.SetToNamecheapDNS(domainName)
		if err != nil {
			return fmt.Errorf("failed to set to provider DNS: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]string{
				jsonKeyDomain: domainName,
				jsonKeyStatus: "reset-to-provider-dns",
			})
		}

		fmt.Printf("Successfully set %s to use provider DNS servers.\n", domainName)
		return nil
	},
}

// domainRenewCmd represents the domain renew command
var domainRenewCmd = &cobra.Command{
	Use:   "renew <domain> [years]",
	Short: "Renew a domain",
	Long:  `Renew a domain for the specified number of years (default: 1 year).`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		years := 1

		if len(args) > 1 {
			var err error
			if years, err = parseYears(args[1]); err != nil {
				return fmt.Errorf("invalid years value: %w", err)
			}
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

		domainService := domain.NewService(client)
		err = domainService.RenewDomain(domainName, years)
		if err != nil {
			return fmt.Errorf("failed to renew domain: %w", err)
		}

		fmt.Printf("Successfully renewed %s for %d year(s).\n", domainName, years)
		return nil
	},
}

func parseYears(yearsStr string) (int, error) {
	var years int
	_, err := fmt.Sscanf(yearsStr, "%d", &years)
	if err != nil {
		return 0, err
	}
	if years < 1 || years > 10 {
		return 0, fmt.Errorf("years must be between 1 and 10")
	}
	return years, nil
}

func init() {
	rootCmd.AddCommand(domainCmd)
	domainCmd.AddCommand(domainListCmd)
	domainCmd.AddCommand(domainInfoCmd)
	domainCmd.AddCommand(domainCheckCmd)
	domainCmd.AddCommand(domainNameserversCmd)
	domainCmd.AddCommand(domainRenewCmd)

	domainNameserversCmd.AddCommand(domainNameserversGetCmd)
	domainNameserversCmd.AddCommand(domainNameserversSetCmd)
	domainNameserversCmd.AddCommand(domainNameserversDefaultCmd)

	domainNameserversSetCmd.Flags().Bool("dry-run", false, "Show what would be set without making any API writes")
	domainNameserversDefaultCmd.Flags().Bool("dry-run", false, "Show what would change without making any API writes")
}
