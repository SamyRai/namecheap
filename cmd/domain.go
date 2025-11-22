package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"namecheap-dns-manager/internal/cmdutil"
	"namecheap-dns-manager/pkg/domain"
)

// domainCmd represents the domain command
var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage Namecheap domains",
	Long:  `Commands for managing Namecheap domains including listing, checking availability, and basic domain operations.`,
}

// domainListCmd represents the domain list command
var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all domains",
	Long:  `List all domains in your Namecheap account with their details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
				dns = "Namecheap"
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

		fmt.Printf("Domain: %s\n", domainInfo.Name)
		fmt.Printf("Owner: %s\n", domainInfo.User)
		fmt.Printf("Created: %s\n", domainInfo.Created)
		fmt.Printf("Expires: %s\n", domainInfo.Expires)
		fmt.Printf("Auto-Renew: %t\n", domainInfo.AutoRenew)
		fmt.Printf("Locked: %t\n", domainInfo.IsLocked)
		fmt.Printf("WhoisGuard: %s\n", domainInfo.WhoisGuard)
		fmt.Printf("Premium: %t\n", domainInfo.IsPremium)
		fmt.Printf("Using Namecheap DNS: %t\n", domainInfo.IsOurDNS)

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
		err = domainService.SetNameservers(domainName, nameservers)
		if err != nil {
			return fmt.Errorf("failed to set nameservers: %w", err)
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
	Short: "Set domain to use Namecheap DNS",
	Long:  `Set the domain to use Namecheap's default DNS servers.`,
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
		err = domainService.SetToNamecheapDNS(domainName)
		if err != nil {
			return fmt.Errorf("failed to set to Namecheap DNS: %w", err)
		}

		fmt.Printf("Successfully set %s to use Namecheap DNS servers.\n", domainName)
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
}
