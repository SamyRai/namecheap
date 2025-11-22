package migadu

import (
	"fmt"
	"strings"

	"namecheap-dns-manager/pkg/dns"
	"namecheap-dns-manager/pkg/plugin"
)

const (
	pluginName        = "migadu"
	pluginVersion     = "1.0.0"
	pluginDescription = "Migadu email hosting setup and management"
)

// MigaduPlugin implements the plugin.Plugin interface for Migadu email hosting
type MigaduPlugin struct{}

// New creates a new Migadu plugin instance
func New() *MigaduPlugin {
	return &MigaduPlugin{}
}

// Name returns the plugin name
func (p *MigaduPlugin) Name() string {
	return pluginName
}

// Description returns the plugin description
func (p *MigaduPlugin) Description() string {
	return pluginDescription
}

// Version returns the plugin version
func (p *MigaduPlugin) Version() string {
	return pluginVersion
}

// Commands returns the list of commands this plugin provides
func (p *MigaduPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "setup",
			Description: "Set up Migadu DNS records for a domain",
			LongDescription: `Set up all necessary DNS records for Migadu email hosting.
This will add:
- MX records for mail routing
- SPF record for sender authentication
- DKIM CNAMEs for email signing
- DMARC record for email policy
- Autoconfig CNAME for email client setup`,
			Execute: func(ctx *plugin.Context) error { return p.setup(ctx) },
		},
		{
			Name:            "verify",
			Description:     "Verify Migadu DNS records for a domain",
			LongDescription: "Check if all required Migadu DNS records are properly configured.",
			Execute:         p.verify,
		},
		{
			Name:            "remove",
			Description:     "Remove Migadu DNS records from a domain",
			LongDescription: "Remove all Migadu-related DNS records from the specified domain.",
			Execute:         func(ctx *plugin.Context) error { return p.remove(ctx) },
		},
	}
}

// setup implements the setup command
func (p *MigaduPlugin) setup(ctx *plugin.Context) error {
	dryRun, _ := ctx.Flags["dry-run"].(bool)
	replace, _ := ctx.Flags["replace"].(bool)

	// Get current records if not replacing
	var existingRecords []dns.Record
	var err error
	if !replace {
		existingRecords, err = ctx.DNS.GetRecords(ctx.Domain)
		if err != nil {
			return fmt.Errorf("failed to get existing records: %w", err)
		}
	}

	// Generate Migadu DNS records
	migaduRecords := p.generateRecords(ctx.Domain)

	ctx.Output.Printf("Setting up Migadu DNS records for %s\n", ctx.Domain)
	ctx.Output.Println("=====================================")

	if dryRun {
		ctx.Output.Println("DRY RUN MODE - No changes will be made")
		ctx.Output.Println()
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
		ctx.Output.Println("⚠️  Conflicting records found:")
		for _, conflict := range conflicts {
			ctx.Output.Printf("   - %s\n", conflict)
		}
		ctx.Output.Println()
		ctx.Output.Println("Use --replace to overwrite existing records or resolve conflicts manually.")
		return nil
	}

	// Show what will be added
	ctx.Output.Println("Records to be added:")
	for _, record := range migaduRecords {
		mxPref := ""
		if record.MXPref > 0 {
			mxPref = fmt.Sprintf(" (priority: %d)", record.MXPref)
		}
		ctx.Output.Printf("  %s %s → %s%s\n", record.HostName, record.RecordType, record.Address, mxPref)
	}
	ctx.Output.Println()

	if dryRun {
		ctx.Output.Println("Dry run completed. Use without --dry-run to apply changes.")
		return nil
	}

	// Apply changes
	var allRecords []dns.Record
	if replace {
		allRecords = migaduRecords
	} else {
		allRecords = existingRecords
		allRecords = append(allRecords, migaduRecords...)
	}

	err = ctx.DNS.SetRecords(ctx.Domain, allRecords)
	if err != nil {
		return fmt.Errorf("failed to set DNS records: %w", err)
	}

	ctx.Output.Printf("✅ Successfully set up Migadu DNS records for %s\n", ctx.Domain)
	ctx.Output.Println()
	ctx.Output.Println("Next steps:")
	ctx.Output.Printf("1. Add %s to your Migadu account\n", ctx.Domain)
	ctx.Output.Println("2. Verify domain ownership in Migadu dashboard")
	ctx.Output.Println("3. Create email accounts in Migadu")
	ctx.Output.Println("4. Test email sending/receiving")

	return nil
}

// verify implements the verify command
func (p *MigaduPlugin) verify(ctx *plugin.Context) error {
	records, err := ctx.DNS.GetRecords(ctx.Domain)
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}

	ctx.Output.Printf("Verifying Migadu setup for %s\n", ctx.Domain)
	ctx.Output.Println("=====================================")

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
		ctx.Output.Printf("%s %s\n", status, check.name)
	}

	ctx.Output.Println()
	if allGood {
		ctx.Output.Println("🎉 All Migadu DNS records are properly configured!")
	} else {
		ctx.Output.Println("⚠️  Some required records are missing or incorrect.")
		ctx.Output.Printf("   Run 'namecheap-dns plugin migadu setup %s' to fix issues.\n", ctx.Domain)
	}

	return nil
}

// remove implements the remove command
func (p *MigaduPlugin) remove(ctx *plugin.Context) error {
	confirm, _ := ctx.Flags["confirm"].(bool)

	if !confirm {
		ctx.Output.Printf("This will remove all Migadu DNS records from %s.\n", ctx.Domain)
		ctx.Output.Println("Use --confirm to proceed.")
		return nil
	}

	records, err := ctx.DNS.GetRecords(ctx.Domain)
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
		ctx.Output.Printf("No Migadu DNS records found for %s\n", ctx.Domain)
		return nil
	}

	err = ctx.DNS.SetRecords(ctx.Domain, filteredRecords)
	if err != nil {
		return fmt.Errorf("failed to update DNS records: %w", err)
	}

	ctx.Output.Printf("✅ Successfully removed %d Migadu DNS records from %s\n", removedCount, ctx.Domain)
	return nil
}

// generateRecords generates the DNS records required for Migadu
func (p *MigaduPlugin) generateRecords(domainName string) []dns.Record {
	return []dns.Record{
		// MX Records
		{
			HostName:   "@",
			RecordType: dns.RecordTypeMX,
			Address:    "aspmx1.migadu.com.",
			TTL:        dns.DefaultTTL,
			MXPref:     10,
		},
		{
			HostName:   "@",
			RecordType: dns.RecordTypeMX,
			Address:    "aspmx2.migadu.com.",
			TTL:        dns.DefaultTTL,
			MXPref:     20,
		},
		// SPF Record
		{
			HostName:   "@",
			RecordType: dns.RecordTypeTXT,
			Address:    "v=spf1 include:spf.migadu.com -all",
			TTL:        dns.DefaultTTL,
		},
		// DMARC Record
		{
			HostName:   "@",
			RecordType: dns.RecordTypeTXT,
			Address:    "v=DMARC1; p=quarantine;",
			TTL:        dns.DefaultTTL,
		},
		// DKIM CNAMEs
		{
			HostName:   "key1._domainkey",
			RecordType: dns.RecordTypeCNAME,
			Address:    fmt.Sprintf("key1.%s._domainkey.migadu.com.", domainName),
			TTL:        dns.DefaultTTL,
		},
		{
			HostName:   "key2._domainkey",
			RecordType: dns.RecordTypeCNAME,
			Address:    fmt.Sprintf("key2.%s._domainkey.migadu.com.", domainName),
			TTL:        dns.DefaultTTL,
		},
		{
			HostName:   "key3._domainkey",
			RecordType: dns.RecordTypeCNAME,
			Address:    fmt.Sprintf("key3.%s._domainkey.migadu.com.", domainName),
			TTL:        dns.DefaultTTL,
		},
		// Autoconfig for email clients
		{
			HostName:   "autoconfig",
			RecordType: dns.RecordTypeCNAME,
			Address:    "autoconfig.migadu.com.",
			TTL:        dns.DefaultTTL,
		},
	}
}
