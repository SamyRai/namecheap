package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"namecheap-dns-manager/internal/cmdutil"
	"namecheap-dns-manager/pkg/dns"
)

// migaduCmd represents the migadu command
var migaduCmd = &cobra.Command{
	Use:   "migadu",
	Short: "Migadu email hosting setup helpers",
	Long:  `Commands for easily setting up Migadu email hosting DNS records.`,
}

// migaduSetupCmd represents the migadu setup command
var migaduSetupCmd = &cobra.Command{
	Use:   "setup <domain>",
	Short: "Set up Migadu DNS records for a domain",
	Long: `Set up all necessary DNS records for Migadu email hosting.
This will add:
- MX records for mail routing
- SPF record for sender authentication
- DKIM CNAMEs for email signing
- DMARC record for email policy
- Autoconfig CNAME for email client setup`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		replace, _ := cmd.Flags().GetBool("replace")

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

		dnsService := dns.NewService(client)

		// Get current records if not replacing
		var existingRecords []dns.Record
		if !replace {
			existingRecords, err = dnsService.GetRecords(domainName)
			if err != nil {
				return fmt.Errorf("failed to get existing records: %w", err)
			}
		}

		// Define Migadu DNS records
		migaduRecords := []dns.Record{
			// MX Records
			{
				HostName:   "@",
				RecordType: "MX",
				Address:    "aspmx1.migadu.com.",
				TTL:        1800,
				MXPref:     10,
			},
			{
				HostName:   "@",
				RecordType: "MX",
				Address:    "aspmx2.migadu.com.",
				TTL:        1800,
				MXPref:     20,
			},
			// SPF Record
			{
				HostName:   "@",
				RecordType: "TXT",
				Address:    "v=spf1 include:spf.migadu.com -all",
				TTL:        1800,
			},
			// DMARC Record
			{
				HostName:   "@",
				RecordType: "TXT",
				Address:    "v=DMARC1; p=quarantine;",
				TTL:        1800,
			},
			// DKIM CNAMEs
			{
				HostName:   "key1._domainkey",
				RecordType: "CNAME",
				Address:    fmt.Sprintf("key1.%s._domainkey.migadu.com.", domainName),
				TTL:        1800,
			},
			{
				HostName:   "key2._domainkey",
				RecordType: "CNAME",
				Address:    fmt.Sprintf("key2.%s._domainkey.migadu.com.", domainName),
				TTL:        1800,
			},
			{
				HostName:   "key3._domainkey",
				RecordType: "CNAME",
				Address:    fmt.Sprintf("key3.%s._domainkey.migadu.com.", domainName),
				TTL:        1800,
			},
			// Autoconfig for email clients
			{
				HostName:   "autoconfig",
				RecordType: "CNAME",
				Address:    "autoconfig.migadu.com.",
				TTL:        1800,
			},
		}

		fmt.Printf("Setting up Migadu DNS records for %s\n", domainName)
		fmt.Println("=====================================")

		if dryRun {
			fmt.Println("DRY RUN MODE - No changes will be made")
			fmt.Println()
		}

		// Check for conflicts if not replacing
		var conflicts []string
		if !replace && len(existingRecords) > 0 {
			for _, migaduRecord := range migaduRecords {
				for _, existing := range existingRecords {
					if existing.HostName == migaduRecord.HostName && existing.RecordType == migaduRecord.RecordType {
						conflicts = append(conflicts, fmt.Sprintf("%s %s", existing.HostName, existing.RecordType))
					}
				}
			}
		}

		if len(conflicts) > 0 && !replace {
			fmt.Println("⚠️  Conflicting records found:")
			for _, conflict := range conflicts {
				fmt.Printf("   - %s\n", conflict)
			}
			fmt.Println()
			fmt.Println("Use --replace to overwrite existing records or resolve conflicts manually.")
			return nil
		}

		// Show what will be added
		fmt.Println("Records to be added:")
		for _, record := range migaduRecords {
			mxPref := ""
			if record.MXPref > 0 {
				mxPref = fmt.Sprintf(" (priority: %d)", record.MXPref)
			}
			fmt.Printf("  %s %s → %s%s\n", record.HostName, record.RecordType, record.Address, mxPref)
		}
		fmt.Println()

		if dryRun {
			fmt.Println("Dry run completed. Use without --dry-run to apply changes.")
			return nil
		}

		// Apply changes
		var allRecords []dns.Record
		if replace {
			allRecords = migaduRecords
		} else {
			// Keep existing records and add Migadu records
			allRecords = existingRecords
			allRecords = append(allRecords, migaduRecords...)
		}

		err = dnsService.SetRecords(domainName, allRecords)
		if err != nil {
			return fmt.Errorf("failed to set DNS records: %w", err)
		}

		fmt.Printf("✅ Successfully set up Migadu DNS records for %s\n", domainName)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("1. Add %s to your Migadu account\n", domainName)
		fmt.Println("2. Verify domain ownership in Migadu dashboard")
		fmt.Println("3. Create email accounts in Migadu")
		fmt.Println("4. Test email sending/receiving")

		return nil
	},
}

// migaduVerifyCmd represents the migadu verify command
var migaduVerifyCmd = &cobra.Command{
	Use:   "verify <domain>",
	Short: "Verify Migadu DNS records for a domain",
	Long:  `Check if all required Migadu DNS records are properly configured.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]

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

		dnsService := dns.NewService(client)
		records, err := dnsService.GetRecords(domainName)
		if err != nil {
			return fmt.Errorf("failed to get DNS records: %w", err)
		}

		fmt.Printf("Verifying Migadu setup for %s\n", domainName)
		fmt.Println("=====================================")

		// Required records for verification
		requiredChecks := []struct {
			name       string
			hostname   string
			recordType string
			valueCheck func(string) bool
			found      bool
		}{
			{
				name:       "MX Record (Primary)",
				hostname:   "@",
				recordType: "MX",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "aspmx1.migadu.com")
				},
			},
			{
				name:       "MX Record (Secondary)",
				hostname:   "@",
				recordType: "MX",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "aspmx2.migadu.com")
				},
			},
			{
				name:       "SPF Record",
				hostname:   "@",
				recordType: "TXT",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "include:spf.migadu.com")
				},
			},
			{
				name:       "DMARC Record",
				hostname:   "@",
				recordType: "TXT",
				valueCheck: func(value string) bool {
					return strings.HasPrefix(value, "v=DMARC1")
				},
			},
			{
				name:       "DKIM Key 1",
				hostname:   "key1._domainkey",
				recordType: "CNAME",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "migadu.com")
				},
			},
			{
				name:       "DKIM Key 2",
				hostname:   "key2._domainkey",
				recordType: "CNAME",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "migadu.com")
				},
			},
			{
				name:       "DKIM Key 3",
				hostname:   "key3._domainkey",
				recordType: "CNAME",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "migadu.com")
				},
			},
			{
				name:       "Autoconfig",
				hostname:   "autoconfig",
				recordType: "CNAME",
				valueCheck: func(value string) bool {
					return strings.Contains(value, "autoconfig.migadu.com")
				},
			},
		}

		// Check each required record
		for i := range requiredChecks {
			check := &requiredChecks[i]
			for _, record := range records {
				if record.HostName == check.hostname && record.RecordType == check.recordType {
					if check.valueCheck(record.Address) {
						check.found = true
						break
					}
				}
			}
		}

		// Display results
		allGood := true
		for _, check := range requiredChecks {
			status := "❌"
			if check.found {
				status = "✅"
			} else {
				allGood = false
			}
			fmt.Printf("%s %s\n", status, check.name)
		}

		fmt.Println()
		if allGood {
			fmt.Println("🎉 All Migadu DNS records are properly configured!")
		} else {
			fmt.Println("⚠️  Some required records are missing or incorrect.")
			fmt.Println("   Run 'namecheap-dns migadu setup " + domainName + "' to fix issues.")
		}

		return nil
	},
}

// migaduRemoveCmd represents the migadu remove command
var migaduRemoveCmd = &cobra.Command{
	Use:   "remove <domain>",
	Short: "Remove Migadu DNS records from a domain",
	Long:  `Remove all Migadu-related DNS records from the specified domain.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			fmt.Printf("This will remove all Migadu DNS records from %s.\n", domainName)
			fmt.Println("Use --confirm to proceed.")
			return nil
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

		dnsService := dns.NewService(client)
		records, err := dnsService.GetRecords(domainName)
		if err != nil {
			return fmt.Errorf("failed to get DNS records: %w", err)
		}

		// Filter out Migadu records
		var filteredRecords []dns.Record
		removedCount := 0

		for _, record := range records {
			isMigaduRecord := false

			// Check if this is a Migadu-related record
			if (record.RecordType == "MX" && strings.Contains(record.Address, "migadu.com")) ||
				(record.RecordType == "TXT" && strings.Contains(record.Address, "spf.migadu.com")) ||
				(record.RecordType == "TXT" && strings.HasPrefix(record.Address, "v=DMARC1")) ||
				(record.RecordType == "CNAME" && strings.Contains(record.Address, "migadu.com")) ||
				(record.HostName == "autoconfig" && record.RecordType == "CNAME") ||
				(strings.Contains(record.HostName, "_domainkey")) {
				isMigaduRecord = true
				removedCount++
			}

			if !isMigaduRecord {
				filteredRecords = append(filteredRecords, record)
			}
		}

		if removedCount == 0 {
			fmt.Printf("No Migadu DNS records found for %s\n", domainName)
			return nil
		}

		err = dnsService.SetRecords(domainName, filteredRecords)
		if err != nil {
			return fmt.Errorf("failed to update DNS records: %w", err)
		}

		fmt.Printf("✅ Successfully removed %d Migadu DNS records from %s\n", removedCount, domainName)
		return nil
	},
}

func init() {
	// Migadu is now available as a plugin via: namecheap-dns plugin migadu <command> <domain>
	// Keeping this for backward compatibility, but it's deprecated
	rootCmd.AddCommand(migaduCmd)
	migaduCmd.AddCommand(migaduSetupCmd)
	migaduCmd.AddCommand(migaduVerifyCmd)
	migaduCmd.AddCommand(migaduRemoveCmd)

	// Flags for migadu setup
	migaduSetupCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	migaduSetupCmd.Flags().Bool("replace", false, "Replace all existing DNS records (use with caution)")

	// Flags for migadu remove
	migaduRemoveCmd.Flags().BoolP("confirm", "y", false, "Confirm removal of Migadu records")
}
