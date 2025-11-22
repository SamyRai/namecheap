package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"namecheap-dns-manager/internal/cmdutil"
	"namecheap-dns-manager/pkg/dns"
)

// dnsCmd represents the dns command
var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS records",
	Long:  `Commands for managing DNS records for your domains.`,
}

// dnsListCmd represents the dns list command
var dnsListCmd = &cobra.Command{
	Use:   "list <domain>",
	Short: "List DNS records for a domain",
	Long:  `List all DNS records for the specified domain.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]

		// Validate domain
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		recordType, _ := cmd.Flags().GetString("type")

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

		var records []dns.Record
		if recordType != "" {
			records, err = dnsService.GetRecordsByType(domainName, strings.ToUpper(recordType))
		} else {
			records, err = dnsService.GetRecords(domainName)
		}

		if err != nil {
			return fmt.Errorf("failed to get DNS records: %w", err)
		}

		if len(records) == 0 {
			fmt.Printf("No DNS records found for %s", domainName)
			if recordType != "" {
				fmt.Printf(" (type: %s)", recordType)
			}
			fmt.Println()
			return nil
		}

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "HOSTNAME\tTYPE\tVALUE\tTTL\tMX_PREF")

		for _, record := range records {
			mxPref := ""
			if record.MXPref > 0 {
				mxPref = strconv.Itoa(record.MXPref)
			}

			ttl := ""
			if record.TTL > 0 {
				ttl = strconv.Itoa(record.TTL)
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				record.HostName, record.RecordType, record.Address, ttl, mxPref)
		}

		w.Flush()
		return nil
	},
}

// dnsAddCmd represents the dns add command
var dnsAddCmd = &cobra.Command{
	Use:   "add <domain> <hostname> <type> <value>",
	Short: "Add a DNS record",
	Long:  `Add a new DNS record to the specified domain.`,
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		hostname := args[1]
		recordType := strings.ToUpper(args[2])
		value := args[3]

		// Validate inputs
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}
		if err := dns.ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}

		ttl, _ := cmd.Flags().GetInt("ttl")
		mxPref, _ := cmd.Flags().GetInt("mx-pref")

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

		record := dns.Record{
			HostName:   hostname,
			RecordType: recordType,
			Address:    value,
			TTL:        ttl,
			MXPref:     mxPref,
		}

		dnsService := dns.NewService(client)

		// Validate record
		if err := dnsService.ValidateRecord(record); err != nil {
			return fmt.Errorf("invalid record: %w", err)
		}

		err = dnsService.AddRecord(domainName, record)
		if err != nil {
			return fmt.Errorf("failed to add DNS record: %w", err)
		}

		fmt.Printf("Successfully added %s record: %s -> %s\n", recordType, hostname, value)
		return nil
	},
}

// dnsUpdateCmd represents the dns update command
var dnsUpdateCmd = &cobra.Command{
	Use:   "update <domain> <hostname> <type> <new-value>",
	Short: "Update a DNS record",
	Long:  `Update an existing DNS record.`,
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		hostname := args[1]
		recordType := strings.ToUpper(args[2])
		newValue := args[3]

		// Validate inputs
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}
		if err := dns.ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}

		ttl, _ := cmd.Flags().GetInt("ttl")
		mxPref, _ := cmd.Flags().GetInt("mx-pref")

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

		newRecord := dns.Record{
			HostName:   hostname,
			RecordType: recordType,
			Address:    newValue,
			TTL:        ttl,
			MXPref:     mxPref,
		}

		dnsService := dns.NewService(client)

		// Validate record
		if err := dnsService.ValidateRecord(newRecord); err != nil {
			return fmt.Errorf("invalid record: %w", err)
		}

		err = dnsService.UpdateRecord(domainName, hostname, recordType, newRecord)
		if err != nil {
			return fmt.Errorf("failed to update DNS record: %w", err)
		}

		fmt.Printf("Successfully updated %s record: %s -> %s\n", recordType, hostname, newValue)
		return nil
	},
}

// dnsDeleteCmd represents the dns delete command
var dnsDeleteCmd = &cobra.Command{
	Use:   "delete <domain> <hostname> <type>",
	Short: "Delete a DNS record",
	Long:  `Delete a DNS record from the specified domain.`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		hostname := args[1]
		recordType := strings.ToUpper(args[2])

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
		err = dnsService.DeleteRecord(domainName, hostname, recordType)
		if err != nil {
			return fmt.Errorf("failed to delete DNS record: %w", err)
		}

		fmt.Printf("Successfully deleted %s record: %s\n", recordType, hostname)
		return nil
	},
}

// dnsClearCmd represents the dns clear command
var dnsClearCmd = &cobra.Command{
	Use:   "clear <domain>",
	Short: "Clear all DNS records for a domain",
	Long:  `Remove all DNS records from the specified domain. Use with caution!`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]

		// Validate domain
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Printf("This will delete ALL DNS records for %s. Use --confirm to proceed.\n", domainName)
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
		err = dnsService.DeleteAllRecords(domainName)
		if err != nil {
			return fmt.Errorf("failed to clear DNS records: %w", err)
		}

		fmt.Printf("Successfully cleared all DNS records for %s\n", domainName)
		return nil
	},
}

// dnsBulkCmd represents the dns bulk command
var dnsBulkCmd = &cobra.Command{
	Use:   "bulk <domain> <operations-file>",
	Short: "Perform bulk DNS operations from a file",
	Long: `Perform multiple DNS operations from a YAML file.

Example file format:
operations:
  - action: add
    hostname: www
    type: A
    value: 192.168.1.1
    ttl: 300
  - action: update
    hostname: mail
    type: A
    value: 192.168.1.2
  - action: delete
    hostname: old
    type: CNAME`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		operationsFile := args[1]

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Show which account is being used
		fmt.Printf("Using account: %s (%s)\n", accountConfig.Username, accountConfig.Description)
		fmt.Println()

		// TODO: Implement bulk operations from file
		// This would involve:
		// 1. Reading and parsing the YAML file
		// 2. Converting to BulkOperation structs
		// 3. Calling dnsService.BulkUpdate()

		return fmt.Errorf("bulk operations not yet implemented - TODO: parse %s and apply to %s", operationsFile, domainName)
	},
}

// dnsImportCmd represents the dns import command
var dnsImportCmd = &cobra.Command{
	Use:   "import <domain> <zone-file>",
	Short: "Import DNS records from a zone file",
	Long:  `Import DNS records from a standard DNS zone file format.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		zoneFile := args[1]

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Show which account is being used
		fmt.Printf("Using account: %s (%s)\n", accountConfig.Username, accountConfig.Description)
		fmt.Println()

		// TODO: Implement zone file import
		// This would involve:
		// 1. Parsing the zone file format
		// 2. Converting to DNS records
		// 3. Setting all records at once

		return fmt.Errorf("zone file import not yet implemented - TODO: parse %s and import to %s", zoneFile, domainName)
	},
}

// dnsExportCmd represents the dns export command
var dnsExportCmd = &cobra.Command{
	Use:   "export <domain> [output-file]",
	Short: "Export DNS records to a zone file",
	Long:  `Export all DNS records to a standard DNS zone file format.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		outputFile := ""
		if len(args) > 1 {
			outputFile = args[1]
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

		// TODO: Implement zone file export
		// This would involve:
		// 1. Converting records to zone file format
		// 2. Writing to file or stdout

		fmt.Printf("Export %d records from %s", len(records), domainName)
		if outputFile != "" {
			fmt.Printf(" to %s", outputFile)
		}
		fmt.Println()
		fmt.Println("TODO: Implement zone file export")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dnsCmd)
	dnsCmd.AddCommand(dnsListCmd)
	dnsCmd.AddCommand(dnsAddCmd)
	dnsCmd.AddCommand(dnsUpdateCmd)
	dnsCmd.AddCommand(dnsDeleteCmd)
	dnsCmd.AddCommand(dnsClearCmd)
	dnsCmd.AddCommand(dnsBulkCmd)
	dnsCmd.AddCommand(dnsImportCmd)
	dnsCmd.AddCommand(dnsExportCmd)

	// Flags for dns list
	dnsListCmd.Flags().StringP("type", "t", "", "Filter by record type (A, AAAA, CNAME, MX, TXT, etc.)")

	// Flags for dns add
	dnsAddCmd.Flags().IntP("ttl", "", 0, "TTL value (Time To Live)")
	dnsAddCmd.Flags().IntP("mx-pref", "", 0, "MX preference value (for MX records)")

	// Flags for dns update
	dnsUpdateCmd.Flags().IntP("ttl", "", 0, "TTL value (Time To Live)")
	dnsUpdateCmd.Flags().IntP("mx-pref", "", 0, "MX preference value (for MX records)")

	// Flags for dns clear
	dnsClearCmd.Flags().BoolP("confirm", "y", false, "Confirm deletion of all records")
}
