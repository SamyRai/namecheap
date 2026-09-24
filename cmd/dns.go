package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"zonekit/internal/cmdutil"
	"zonekit/pkg/dns"
	"zonekit/pkg/dns/zonefile"
	"zonekit/pkg/dnsrecord"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newDNSService builds the DNS service used by every command in this file.
// It is a package variable (not a direct dns.NewService call) so tests can
// substitute a service backed by a mock provider instead of the real
// Namecheap client, without any network access.
var newDNSService = dns.NewService

// JSON/YAML result map keys reused across several commands in this file.
const (
	jsonKeyStatus = "status"
	jsonKeyDomain = "domain"
)

// dnsPlanDTO is the stable wire shape for a dns.PlannedChange in --dry-run
// json/yaml output, reusing cmdutil.RecordDTO's field names for before/after
// so a plan's records look exactly like a `dns list` record.
type dnsPlanDTO struct {
	Action string             `json:"action" yaml:"action"`
	Before *cmdutil.RecordDTO `json:"before,omitempty" yaml:"before,omitempty"`
	After  *cmdutil.RecordDTO `json:"after,omitempty" yaml:"after,omitempty"`
}

func newDNSPlanDTO(p dns.PlannedChange) dnsPlanDTO {
	dto := dnsPlanDTO{Action: p.Action}
	if p.Before != nil {
		b := cmdutil.NewRecordDTO(*p.Before)
		dto.Before = &b
	}
	if p.After != nil {
		a := cmdutil.NewRecordDTO(*p.After)
		dto.After = &a
	}
	return dto
}

// printPlannedChanges renders a --dry-run preview and performs no provider
// calls of its own (the plans passed in must already have been produced by
// one of the Service.Plan* methods, which are read-only). In json/yaml mode
// it writes the list of changes as data; in table mode it prints one
// human-readable line per change.
func printPlannedChanges(format string, plans []dns.PlannedChange) error {
	if format != cmdutil.OutputTable {
		dtos := make([]dnsPlanDTO, len(plans))
		for i, p := range plans {
			dtos[i] = newDNSPlanDTO(p)
		}
		return cmdutil.WriteResult(os.Stdout, format, dtos)
	}

	fmt.Printf("DRY RUN: %d change(s) would be made. No records were modified.\n", len(plans))
	for _, p := range plans {
		fmt.Println(formatPlannedChangeLine(p))
	}
	return nil
}

func formatPlannedChangeLine(p dns.PlannedChange) string {
	switch p.Action {
	case dns.PlanActionCreate:
		r := p.After
		return fmt.Sprintf("  + create %s\t%s\t%s", r.HostName, r.RecordType, r.Address)
	case dns.PlanActionUpdate:
		return fmt.Sprintf("  ~ update %s\t%s\t%s -> %s", p.After.HostName, p.After.RecordType, p.Before.Address, p.After.Address)
	case dns.PlanActionDelete:
		r := p.Before
		return fmt.Sprintf("  - delete %s\t%s\t%s", r.HostName, r.RecordType, r.Address)
	default:
		return fmt.Sprintf("  ? %s", p.Action)
	}
}

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
		format := OutputFormat()

		// Validate domain
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		recordType, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		dnsService := newDNSService(ncClient)

		var records []dnsrecord.Record
		if recordType != "" {
			records, err = dnsService.GetRecordsByType(domainName, strings.ToUpper(recordType))
		} else {
			records, err = dnsService.GetRecords(domainName)
		}

		if err != nil {
			return fmt.Errorf("failed to get DNS records: %w", err)
		}

		if name != "" {
			records = filterRecordsByExactName(records, name)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, cmdutil.NewRecordDTOs(records))
		}

		if len(records) == 0 {
			fmt.Printf("No DNS records found for %s", domainName)
			if recordType != "" {
				fmt.Printf(" (type: %s)", recordType)
			}
			if name != "" {
				fmt.Printf(" (name: %s)", name)
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

// capitalize upper-cases the first rune of s. strings.Title is deprecated
// (it Unicode-title-cases every word); bulk operation actions are always a
// single lowercase ASCII word ("add"/"update"/"delete"), so this is enough.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// filterRecordsByExactName keeps only records whose hostname matches name
// exactly (case-insensitively, matching DNS convention). Unlike --type, this
// is never a substring match (O5): "www" must not also return "www2".
func filterRecordsByExactName(records []dnsrecord.Record, name string) []dnsrecord.Record {
	filtered := make([]dnsrecord.Record, 0, len(records))
	for _, r := range records {
		if strings.EqualFold(r.HostName, name) {
			filtered = append(filtered, r)
		}
	}
	return filtered
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
		format := OutputFormat()

		// Validate inputs
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}
		if err := dns.ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}

		ttl, _ := cmd.Flags().GetInt("ttl")
		mxPref, _ := cmd.Flags().GetInt("mx-pref")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		record := dnsrecord.Record{
			HostName:   hostname,
			RecordType: recordType,
			Address:    value,
			TTL:        ttl,
			MXPref:     mxPref,
		}

		dnsService := newDNSService(ncClient)

		// Validate record
		if err := dnsService.ValidateRecord(record); err != nil {
			return fmt.Errorf("invalid record: %w", err)
		}

		if dryRun {
			plan, err := dnsService.PlanAddRecord(record)
			if err != nil {
				return fmt.Errorf("failed to plan DNS record add: %w", err)
			}
			return printPlannedChanges(format, []dns.PlannedChange{plan})
		}

		err = dnsService.AddRecord(domainName, record)
		if err != nil {
			return fmt.Errorf("failed to add DNS record: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, cmdutil.NewRecordDTO(record))
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
		format := OutputFormat()

		// Validate inputs
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}
		if err := dns.ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}

		ttl, _ := cmd.Flags().GetInt("ttl")
		mxPref, _ := cmd.Flags().GetInt("mx-pref")
		matchValue, _ := cmd.Flags().GetString("match-value")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		newRecord := dnsrecord.Record{
			HostName:   hostname,
			RecordType: recordType,
			Address:    newValue,
			TTL:        ttl,
			MXPref:     mxPref,
		}

		dnsService := newDNSService(ncClient)

		// Validate record
		if err := dnsService.ValidateRecord(newRecord); err != nil {
			return fmt.Errorf("invalid record: %w", err)
		}

		if dryRun {
			plan, err := dnsService.PlanUpdateRecordMatching(domainName, hostname, recordType, matchValue, newRecord)
			if err != nil {
				return fmt.Errorf("failed to plan DNS record update: %w", err)
			}
			return printPlannedChanges(format, []dns.PlannedChange{plan})
		}

		err = dnsService.UpdateRecordMatching(domainName, hostname, recordType, matchValue, newRecord)
		if err != nil {
			return fmt.Errorf("failed to update DNS record: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, cmdutil.NewRecordDTO(newRecord))
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
		format := OutputFormat()
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		dnsService := newDNSService(ncClient)

		if dryRun {
			plan, err := dnsService.PlanDeleteRecord(domainName, hostname, recordType)
			if err != nil {
				return fmt.Errorf("failed to plan DNS record delete: %w", err)
			}
			return printPlannedChanges(format, []dns.PlannedChange{plan})
		}

		err = dnsService.DeleteRecord(domainName, hostname, recordType)
		if err != nil {
			return fmt.Errorf("failed to delete DNS record: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]string{
				"hostname":    hostname,
				"type":        recordType,
				jsonKeyStatus: "deleted",
			})
		}

		fmt.Printf("Successfully deleted %s record: %s\n", recordType, hostname)
		return nil
	},
}

// dnsClearCmd represents the dns clear command
var dnsClearCmd = &cobra.Command{
	Use:   "clear <domain>",
	Short: "Clear all DNS records for a domain",
	Long: `Remove all DNS records from the specified domain. Use with caution!

Requires --confirm (refused otherwise, even with --dry-run). Combine with
--dry-run to preview which records would be removed without deleting them.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		format := OutputFormat()
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Validate domain
		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		dnsService := newDNSService(ncClient)

		if dryRun {
			plans, err := dnsService.PlanClear(domainName)
			if err != nil {
				return fmt.Errorf("failed to plan DNS clear: %w", err)
			}
			return printPlannedChanges(format, plans)
		}

		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			return fmt.Errorf("this would delete ALL DNS records for %s; refusing without --confirm (use --dry-run to preview first)", domainName)
		}

		err = dnsService.DeleteAllRecords(domainName)
		if err != nil {
			return fmt.Errorf("failed to clear DNS records: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]string{
				jsonKeyDomain: domainName,
				jsonKeyStatus: "cleared",
			})
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
		format := OutputFormat()
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Get current account configuration
		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		// Create client and display account info
		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		dnsService := newDNSService(ncClient)

		// Parse the operations file
		operations, err := parseBulkOperationsFile(operationsFile)
		if err != nil {
			return fmt.Errorf("failed to parse operations file: %w", err)
		}

		if len(operations) == 0 {
			return fmt.Errorf("no operations found in file %s", operationsFile)
		}

		if dryRun {
			plans, err := dnsService.PlanBulkUpdate(domainName, operations)
			if err != nil {
				return fmt.Errorf("failed to plan bulk operations: %w", err)
			}
			return printPlannedChanges(format, plans)
		}

		if format == cmdutil.OutputTable {
			// Show what will be done
			fmt.Printf("Applying %d bulk operations to %s\n", len(operations), domainName)
			fmt.Println("=====================================")

			for i, op := range operations {
				action := capitalize(op.Action)
				fmt.Printf("%d. %s %s %s → %s", i+1, action, op.Record.HostName, op.Record.RecordType, op.Record.Address)
				if op.Record.TTL > 0 {
					fmt.Printf(" (TTL: %d)", op.Record.TTL)
				}
				if op.Record.MXPref > 0 {
					fmt.Printf(" (Priority: %d)", op.Record.MXPref)
				}
				fmt.Println()
			}
			fmt.Println()
		}

		// Confirm before proceeding
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			if format != cmdutil.OutputTable {
				return fmt.Errorf("%d bulk operations require --confirm to apply (use --dry-run to preview)", len(operations))
			}
			fmt.Println("Use --confirm to apply these changes.")
			return nil
		}

		// Apply the operations
		err = dnsService.BulkUpdate(domainName, operations)
		if err != nil {
			return fmt.Errorf("failed to apply bulk operations: %w", err)
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]interface{}{
				jsonKeyDomain: domainName,
				"operations":  len(operations),
				jsonKeyStatus: "applied",
			})
		}

		fmt.Printf("✅ Successfully applied %d bulk operations to %s\n", len(operations), domainName)
		return nil
	},
}

// dnsEnsureCmd represents the dns ensure command (O4): an idempotent
// create-or-update, keyed by hostname+type (+ value for MX/TXT — see
// dns.Service.Ensure's doc comment for the exact identity rule).
var dnsEnsureCmd = &cobra.Command{
	Use:   "ensure <domain> <hostname> <type> <value>",
	Short: "Idempotently create or update a DNS record",
	Long: `Ensure a DNS record with the given hostname, type, and value exists,
creating or updating it as needed. Running the same command twice reports
"unchanged" the second time.

Identity rule: for MX and TXT records, (hostname, type, value) identifies the
record, since a hostname can carry several of each at once (multiple MX
priorities, SPF/DMARC/verification TXT records); a non-matching value is
added alongside the others rather than overwriting one of them. For every
other type, (hostname, type) identifies the record; if more than one already
exists, ensure refuses rather than guess which to update.`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		hostname := args[1]
		recordType := strings.ToUpper(args[2])
		value := args[3]
		format := OutputFormat()

		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}
		if err := dns.ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}

		ttl, _ := cmd.Flags().GetInt("ttl")
		mxPref, _ := cmd.Flags().GetInt("mx-pref")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}

		ncClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)

		record := dnsrecord.Record{
			HostName:   hostname,
			RecordType: recordType,
			Address:    value,
			TTL:        ttl,
			MXPref:     mxPref,
		}

		dnsService := newDNSService(ncClient)

		var result dns.EnsureResult
		if dryRun {
			result, err = dnsService.PlanEnsure(domainName, record)
			if err != nil {
				return fmt.Errorf("failed to plan DNS ensure: %w", err)
			}
		} else {
			result, err = dnsService.Ensure(domainName, record)
			if err != nil {
				return fmt.Errorf("failed to ensure DNS record: %w", err)
			}
		}

		if format != cmdutil.OutputTable {
			return cmdutil.WriteResult(os.Stdout, format, map[string]interface{}{
				jsonKeyStatus: result.Status,
				"record":      cmdutil.NewRecordDTO(result.Record),
			})
		}

		prefix := ""
		if dryRun {
			prefix = "DRY RUN: would report "
		}
		fmt.Printf("%s%s: %s %s -> %s\n", prefix, result.Status, hostname, recordType, result.Record.Address)
		return nil
	},
}

// dnsImportCmd represents the dns import command
var dnsImportCmd = &cobra.Command{
	Use:   "import <domain> <zone-file>",
	Short: "Import DNS records from a zone file",
	Long: `Import DNS records from a zone file (as produced by "zonekit dns export").

Import REPLACES ALL existing DNS records for the domain with the records
parsed from the file. Review the listed records and pass --confirm to apply
them.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		zoneFile := args[1]

		if err := dns.ValidateDomain(domainName); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		content, err := os.ReadFile(zoneFile)
		if err != nil {
			return fmt.Errorf("failed to read zone file: %w", err)
		}

		records, err := zonefile.Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse zone file %s: %w", zoneFile, err)
		}
		if len(records) == 0 {
			return fmt.Errorf("no records found in %s", zoneFile)
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

		fmt.Printf("This will replace ALL DNS records for %s with %d record(s) from %s:\n", domainName, len(records), zoneFile)
		for _, record := range records {
			fmt.Printf("  %s IN %s %s\n", record.HostName, record.RecordType, record.Address)
		}
		fmt.Println()

		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("Use --confirm to apply these changes.")
			return nil
		}

		dnsService := dns.NewService(client)
		if err := dnsService.SetRecords(domainName, records); err != nil {
			return fmt.Errorf("failed to import DNS records: %w", err)
		}

		fmt.Printf("✅ Successfully imported %d record(s) from %s to %s\n", len(records), zoneFile, domainName)
		return nil
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

		// Convert records to zone file format
		zoneContent := zonefile.Format(domainName, records)

		if outputFile != "" {
			// Write to file
			err = os.WriteFile(outputFile, []byte(zoneContent), 0644)
			if err != nil {
				return fmt.Errorf("failed to write zone file: %w", err)
			}
			fmt.Printf("✅ Exported %d records from %s to %s\n", len(records), domainName, outputFile)
		} else {
			// Write to stdout
			fmt.Printf("Zone file for %s:\n", domainName)
			fmt.Println("=====================================")
			fmt.Print(zoneContent)
		}

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
	dnsCmd.AddCommand(dnsEnsureCmd)
	dnsCmd.AddCommand(dnsImportCmd)
	dnsCmd.AddCommand(dnsExportCmd)

	// Flags for dns list
	dnsListCmd.Flags().StringP("type", "t", "", "Filter by record type (A, AAAA, CNAME, MX, TXT, etc.)")
	dnsListCmd.Flags().StringP("name", "n", "", "Filter by exact hostname (case-insensitive, not a substring match)")

	// Flags for dns add
	dnsAddCmd.Flags().IntP("ttl", "", 0, "TTL value (Time To Live)")
	dnsAddCmd.Flags().IntP("mx-pref", "", 0, "MX preference value (for MX records)")
	dnsAddCmd.Flags().Bool("dry-run", false, "Show what would be added without making any API writes")

	// Flags for dns update
	dnsUpdateCmd.Flags().IntP("ttl", "", 0, "TTL value (Time To Live)")
	dnsUpdateCmd.Flags().IntP("mx-pref", "", 0, "MX preference value (for MX records)")
	dnsUpdateCmd.Flags().StringP("match-value", "", "",
		"Current value of the record to replace. Required when several records share the hostname and type (e.g. multiple TXT at the apex)")
	dnsUpdateCmd.Flags().Bool("dry-run", false, "Show what would change without making any API writes")

	// Flags for dns delete
	dnsDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without making any API writes")

	// Flags for dns clear
	dnsClearCmd.Flags().BoolP("confirm", "y", false, "Confirm deletion of all records (required unless --dry-run)")
	dnsClearCmd.Flags().Bool("dry-run", false, "Show what would be deleted without making any API writes or requiring --confirm")

	// Flags for dns bulk
	dnsBulkCmd.Flags().BoolP("confirm", "y", false, "Confirm the bulk operations")
	dnsBulkCmd.Flags().Bool("dry-run", false, "Show what would change without making any API writes or requiring --confirm")

	// Flags for dns ensure
	dnsEnsureCmd.Flags().IntP("ttl", "", 0, "TTL value (Time To Live)")
	dnsEnsureCmd.Flags().IntP("mx-pref", "", 0, "MX preference value (for MX records)")
	dnsEnsureCmd.Flags().Bool("dry-run", false, "Show what would change (created/updated/unchanged) without making any API writes")

	// Flags for dns import
	dnsImportCmd.Flags().BoolP("confirm", "y", false, "Confirm the import (replaces all existing records)")
}

// parseBulkOperationsFile parses a YAML file containing bulk DNS operations
func parseBulkOperationsFile(filePath string) ([]dns.BulkOperation, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read operations file: %w", err)
	}

	// Define the structure for parsing
	type OperationInput struct {
		Action   string `yaml:"action"`
		Hostname string `yaml:"hostname"`
		Type     string `yaml:"type"`
		Value    string `yaml:"value"`
		TTL      int    `yaml:"ttl,omitempty"`
		MXPref   int    `yaml:"mx_pref,omitempty"`
	}

	var inputs []OperationInput
	if err := yaml.Unmarshal(data, &inputs); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var operations []dns.BulkOperation
	for _, input := range inputs {
		// Validate required fields
		if input.Action == "" {
			return nil, fmt.Errorf("operation missing required field: action")
		}
		if input.Hostname == "" {
			return nil, fmt.Errorf("operation missing required field: hostname")
		}
		if input.Type == "" {
			return nil, fmt.Errorf("operation missing required field: type")
		}
		if input.Value == "" {
			return nil, fmt.Errorf("operation missing required field: value")
		}

		// Validate action
		action := strings.ToLower(input.Action)
		if action != dns.BulkActionAdd && action != dns.BulkActionUpdate && action != dns.BulkActionDelete {
			return nil, fmt.Errorf("invalid action '%s', must be one of: %s, %s, %s", input.Action, dns.BulkActionAdd, dns.BulkActionUpdate, dns.BulkActionDelete)
		}

		// Create the record
		record := dnsrecord.Record{
			HostName:   input.Hostname,
			RecordType: input.Type,
			Address:    input.Value,
			TTL:        input.TTL,
			MXPref:     input.MXPref,
		}

		operation := dns.BulkOperation{
			Action: action,
			Record: record,
		}

		operations = append(operations, operation)
	}

	return operations, nil
}
