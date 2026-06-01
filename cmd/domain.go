package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"zonekit/internal/cmdutil"
	pkgclient "zonekit/pkg/client"
	"zonekit/pkg/config"
	"zonekit/pkg/domain"
	"zonekit/pkg/domain/model"
)

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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		domains, err := domainService.ListDomains(cmd.Context())
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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		domainInfo, err := domainService.GetDomainInfo(cmd.Context(), domainName)
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
		fmt.Printf("Using Provider DNS: %t\n", domainInfo.IsOurDNS)

		return nil
	},
}

// domainRegisterCmd represents the domain register command
var domainRegisterCmd = &cobra.Command{
	Use:   "register <domain>",
	Short: "Register a new domain",
	Long:  `Register a new domain using the configured contact profile.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		
		years, _ := cmd.Flags().GetInt("years")
		profileName, _ := cmd.Flags().GetString("contact-profile")

		// Validate domain
		if err := domain.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		// Get config manager to access ContactProfiles
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to get config manager: %w", err)
		}

		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}
		
		profile, err := configManager.GetContactProfile(profileName)
		if err != nil {
			return fmt.Errorf("failed to load contact profile '%s' (add contact_profiles block to ~/.zonekit.yaml): %w", profileName, err)
		}

		// Create client and display account info
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		
		cInfo := model.ContactInfo{
			FirstName:      profile.FirstName,
			LastName:       profile.LastName,
			Address:        profile.Address,
			City:           profile.City,
			StateProvince:  profile.StateProvince,
			PostalCode:     profile.PostalCode,
			Country:        profile.Country,
			Phone:          profile.Phone,
			Email:          profile.Email,
		}

		req := model.RegistrationRequest{
			DomainName: domainName,
			Years:      years,
			Registrant: cInfo,
			Tech:       cInfo,
			Admin:      cInfo,
			AuxBilling: cInfo,
		}

		err = domainService.RegisterDomain(cmd.Context(), req)
		pkgclient.LogAuditEvent(accountName, accountConfig.Provider, "RegisterDomain", domainName, err)
		if err != nil {
			return fmt.Errorf("failed to register domain: %w", err)
		}

		fmt.Printf("Successfully registered %s for %d year(s).\n", domainName, years)
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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		available, err := domainService.CheckAvailability(cmd.Context(), domainName)
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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		nameservers, err := domainService.GetNameservers(cmd.Context(), domainName)
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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		err = domainService.SetNameservers(cmd.Context(), domainName, nameservers)
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
	Short: "Set domain to use provider DNS",
	Long:  `Set the domain to use the provider's default DNS servers.`,
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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		err = domainService.SetDefaultNameservers(cmd.Context(), domainName)
		if err != nil {
			return fmt.Errorf("failed to set to provider DNS: %w", err)
		}

		fmt.Printf("Successfully set %s to use provider DNS servers.\n", domainName)
		return nil
	},
}

// domainDnssecCmd represents the domain dnssec command
var domainDnssecCmd = &cobra.Command{
	Use:   "dnssec",
	Short: "Manage DNSSEC for a domain",
	Long:  `Enable, disable, or check the status of DNSSEC for a domain.`,
}

var domainDnssecEnableCmd = &cobra.Command{
	Use:   "enable <domain>",
	Short: "Enable DNSSEC",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		
		configManager, err := config.NewManager()
		if err != nil {
			return err
		}
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			return err
		}
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}

		err = domainService.EnableDNSSEC(cmd.Context(), domainName)
		pkgclient.LogAuditEvent(accountConfig.Username, accountConfig.Provider, "EnableDNSSEC", domainName, err)
		if err != nil {
			return fmt.Errorf("failed to enable DNSSEC: %w", err)
		}

		fmt.Printf("Successfully enabled DNSSEC for %s\n", domainName)
		return nil
	},
}

var domainDnssecDisableCmd = &cobra.Command{
	Use:   "disable <domain>",
	Short: "Disable DNSSEC",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		
		configManager, err := config.NewManager()
		if err != nil {
			return err
		}
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			return err
		}
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}

		err = domainService.DisableDNSSEC(cmd.Context(), domainName)
		pkgclient.LogAuditEvent(accountConfig.Username, accountConfig.Provider, "DisableDNSSEC", domainName, err)
		if err != nil {
			return fmt.Errorf("failed to disable DNSSEC: %w", err)
		}

		fmt.Printf("Successfully disabled DNSSEC for %s\n", domainName)
		return nil
	},
}

var domainDnssecStatusCmd = &cobra.Command{
	Use:   "status <domain>",
	Short: "Get DNSSEC status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		
		configManager, err := config.NewManager()
		if err != nil {
			return err
		}
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			return err
		}
		client, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}

		status, err := domainService.GetDNSSECStatus(cmd.Context(), domainName)
		if err != nil {
			return fmt.Errorf("failed to get DNSSEC status: %w", err)
		}

		if status {
			fmt.Printf("DNSSEC is ENABLED for %s\n", domainName)
		} else {
			fmt.Printf("DNSSEC is DISABLED for %s\n", domainName)
		}
		return nil
	},
}
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

		domainService, err := domain.NewService(client)
		if err != nil {
			return fmt.Errorf("failed to create domain service: %w", err)
		}
		err = domainService.RenewDomain(cmd.Context(), domainName, years)
		pkgclient.LogAuditEvent(accountConfig.Username, accountConfig.Provider, "RenewDomain", domainName, err)
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
	domainCmd.AddCommand(domainRegisterCmd)
	domainCmd.AddCommand(domainNameserversCmd)
	domainCmd.AddCommand(domainDnssecCmd)
	domainCmd.AddCommand(domainRenewCmd)

	domainDnssecCmd.AddCommand(domainDnssecEnableCmd)
	domainDnssecCmd.AddCommand(domainDnssecDisableCmd)
	domainDnssecCmd.AddCommand(domainDnssecStatusCmd)

	domainRegisterCmd.Flags().IntP("years", "y", 1, "Number of years to register the domain")
	domainRegisterCmd.Flags().StringP("contact-profile", "p", "default", "Contact profile from config to use for registration")

	domainNameserversCmd.AddCommand(domainNameserversGetCmd)
	domainNameserversCmd.AddCommand(domainNameserversSetCmd)
	domainNameserversCmd.AddCommand(domainNameserversDefaultCmd)
}
