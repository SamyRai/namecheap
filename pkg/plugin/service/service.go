package service

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"zonekit/pkg/dns"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/plugin"
)

// ServicePlugin is a generic plugin that loads service integration configurations
type ServicePlugin struct {
	configs map[string]*Config
}

// NewServicePlugin creates a new service plugin
func NewServicePlugin(configs map[string]*Config) *ServicePlugin {
	return &ServicePlugin{
		configs: configs,
	}
}

// Name returns the plugin name
func (p *ServicePlugin) Name() string {
	return "service"
}

// Description returns the plugin description
func (p *ServicePlugin) Description() string {
	return "DNS record templates for service integrations (email, CDN, hosting, etc.)"
}

// Version returns the plugin version
func (p *ServicePlugin) Version() string {
	return "1.0.0"
}

// Commands returns the list of commands this plugin provides
func (p *ServicePlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "setup",
			Description: "Set up DNS records for a service integration",
			LongDescription: `Set up all necessary DNS records for a configured service integration.
Usage: service setup <service-name> <domain>

Available services can be listed with: service list`,
			Execute: p.setup,
		},
		{
			Name:            "verify",
			Description:     "Verify DNS records for a service integration",
			LongDescription: "Check if all required DNS records for a service integration are properly configured.",
			Execute:         p.verify,
		},
		{
			Name:            "remove",
			Description:     "Remove DNS records for a service integration",
			LongDescription: "Remove all service-related DNS records from the specified domain.",
			Execute:         p.remove,
		},
		{
			Name:            "list",
			Description:     "List all available service integrations",
			LongDescription: "List all configured service integrations.",
			Execute:         p.list,
		},
		{
			Name:            "info",
			Description:     "Show service integration information",
			LongDescription: "Display detailed information about a specific service integration.",
			Execute:         p.info,
		},
	}
}

// supersededBy reports whether an existing zone record is replaced by one of
// the service's records. For most types that is a hostname+type match, but TXT
// is deliberately narrower: TXT is multi-valued, and a zone apex routinely
// carries an SPF policy alongside unrelated ownership proofs
// (google-site-verification, yandex-verification, and so on). Matching TXT on
// hostname+type alone would evict all of them the moment a service publishes
// its SPF record, silently breaking Search Console and similar domain claims.
// So a TXT only supersedes another TXT of the same kind.
func supersededBy(existing dnsrecord.Record, records []dnsrecord.Record) bool {
	for _, r := range records {
		if existing.HostName != r.HostName || existing.RecordType != r.RecordType {
			continue
		}
		if existing.RecordType == dnsrecord.RecordTypeTXT &&
			txtKind(existing.Address) != txtKind(r.Address) {
			continue
		}
		return true
	}
	return false
}

// txtKind classifies a TXT payload by its policy prefix. Unrecognized values
// are keyed by their own content, so an unrelated verification token only ever
// matches an identical one (a harmless de-duplication) and is otherwise kept.
func txtKind(value string) string {
	v := strings.ToLower(strings.TrimSpace(strings.Trim(value, `"`)))
	switch {
	case strings.HasPrefix(v, "v=spf1"):
		return "spf"
	case strings.HasPrefix(v, "v=dmarc1"):
		return "dmarc"
	case strings.HasPrefix(v, "v=dkim1"):
		return "dkim"
	default:
		return "literal:" + v
	}
}

// setup implements the setup command
func (p *ServicePlugin) setup(ctx *plugin.Context) error {
	if len(ctx.Args) < 2 {
		return fmt.Errorf("usage: service setup <service-name> <domain>")
	}

	serviceName := ctx.Args[0]
	domain := ctx.Args[1]

	config, exists := p.configs[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found. Use 'service list' to see available services", serviceName)
	}

	dryRun, _ := ctx.Flags["dry-run"].(bool)
	replace, _ := ctx.Flags["replace"].(bool)

	// Always read the current zone, even with --replace. Providers apply this
	// through a whole-zone write (Namecheap's setHosts replaces the entire
	// record set), so anything missing from the slice we hand back is DELETED.
	// Skipping this fetch under --replace meant publishing only the service's
	// own records, wiping every unrelated A/CNAME/TXT in the zone: website and
	// API hosts, ACME challenges, other services' verification tokens.
	existingRecords, err := ctx.DNS.GetRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	// Generate DNS records from config
	records := p.generateRecords(config, domain)

	ctx.Output.Printf("Setting up %s DNS records for %s\n", config.DisplayName, domain)
	ctx.Output.Println("=====================================")

	if dryRun {
		ctx.Output.Println("DRY RUN MODE - No changes will be made")
		ctx.Output.Println()
	}

	// Split the existing zone into records this service supersedes and records
	// that must survive untouched. Under --replace only the former are dropped.
	var superseded []dnsrecord.Record
	var preserved []dnsrecord.Record
	for _, existing := range existingRecords {
		if supersededBy(existing, records) {
			superseded = append(superseded, existing)
		} else {
			preserved = append(preserved, existing)
		}
	}

	// Check for conflicts if not replacing
	var conflicts []string
	if !replace {
		for _, existing := range superseded {
			conflicts = append(conflicts, fmt.Sprintf("%s %s", existing.HostName, existing.RecordType))
		}
	}

	if len(conflicts) > 0 && !replace {
		ctx.Output.Println("Conflicting records found:")
		for _, conflict := range conflicts {
			ctx.Output.Printf("   - %s\n", conflict)
		}
		ctx.Output.Println()
		ctx.Output.Println("Use --replace to overwrite existing records or resolve conflicts manually.")
		return nil
	}

	// Show what will be added
	ctx.Output.Println("Records to be added:")
	for _, record := range records {
		mxPref := ""
		if record.MXPref > 0 {
			mxPref = fmt.Sprintf(" (priority: %d)", record.MXPref)
		}
		ctx.Output.Printf("  %s %s → %s%s\n", record.HostName, record.RecordType, record.Address, mxPref)
	}
	ctx.Output.Println()

	if replace && len(superseded) > 0 {
		ctx.Output.Println("Records to be replaced:")
		for _, record := range superseded {
			ctx.Output.Printf("  %s %s → %s\n", record.HostName, record.RecordType, record.Address)
		}
		ctx.Output.Println()
	}

	// The zone is rewritten wholesale, so state how much of it is carried
	// through untouched. A silent count is the difference between noticing a
	// wipe and publishing one.
	ctx.Output.Printf("Preserving %d unrelated record(s) in the zone.\n", len(preserved))
	ctx.Output.Println()

	if dryRun {
		ctx.Output.Println("Dry run completed. Use without --dry-run to apply changes.")
		return nil
	}

	// Apply changes. Both paths keep every record the service does not own;
	// --replace differs only in dropping the superseded same-kind ones.
	allRecords := append([]dnsrecord.Record{}, preserved...)
	if !replace {
		allRecords = append(allRecords, superseded...)
	}
	allRecords = append(allRecords, records...)

	err = ctx.DNS.SetRecords(domain, allRecords)
	if err != nil {
		return fmt.Errorf("failed to set DNS records: %w", err)
	}

	ctx.Output.Printf("Successfully set up %s DNS records for %s\n", config.DisplayName, domain)
	ctx.Output.Println()
	ctx.Output.Println("Next steps:")
	ctx.Output.Printf("1. Configure %s in your %s account\n", domain, config.DisplayName)
	ctx.Output.Println("2. Verify domain ownership if required")
	ctx.Output.Println("3. Test the configuration")

	return nil
}

// verify implements the verify command
func (p *ServicePlugin) verify(ctx *plugin.Context) error {
	if len(ctx.Args) < 2 {
		return fmt.Errorf("usage: service verify <service-name> <domain>")
	}

	serviceName := ctx.Args[0]
	domain := ctx.Args[1]

	config, exists := p.configs[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found. Use 'service list' to see available services", serviceName)
	}

	records, err := ctx.DNS.GetRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}

	ctx.Output.Printf("Verifying %s setup for %s\n", config.DisplayName, domain)
	ctx.Output.Println("=====================================")

	// Perform verification checks
	allGood := true
	if config.Verification != nil && len(config.Verification.RequiredRecords) > 0 {
		for _, check := range config.Verification.RequiredRecords {
			found := false
			for _, record := range records {
				if record.HostName == check.Hostname && record.RecordType == check.Type {
					// Check value matches
					matches := false
					if check.Contains != "" {
						matches = strings.Contains(record.Address, check.Contains)
					} else if check.Equals != "" {
						matches = record.Address == check.Equals
					} else if check.StartsWith != "" {
						matches = strings.HasPrefix(record.Address, check.StartsWith)
					} else {
						matches = true // Just check existence
					}

					if matches {
						found = true
						break
					}
				}
			}

			status := "FAIL"
			if found {
				status = "PASS"
			} else {
				allGood = false
			}

			ctx.Output.Printf("%s %s %s (%s)\n", status, check.Type, check.Hostname, check.Type)
		}
	} else {
		// Generic verification - check if generated records exist
		expectedRecords := p.generateRecords(config, domain)
		for _, expected := range expectedRecords {
			found := false
			for _, actual := range records {
				if actual.HostName == expected.HostName &&
					actual.RecordType == expected.RecordType &&
					strings.Contains(actual.Address, strings.TrimSuffix(expected.Address, ".")) {
					found = true
					break
				}
			}

			status := "FAIL"
			if found {
				status = "PASS"
			} else {
				allGood = false
			}

			ctx.Output.Printf("%s %s %s\n", status, expected.RecordType, expected.HostName)
		}
	}

	ctx.Output.Println()
	if allGood {
		ctx.Output.Printf("All %s DNS records are properly configured!\n", config.DisplayName)
	} else {
		ctx.Output.Println("Some required records are missing or incorrect.")
		ctx.Output.Printf("Run 'zonekit service setup %s %s' to fix issues.\n", serviceName, domain)
	}

	return nil
}

// remove implements the remove command
func (p *ServicePlugin) remove(ctx *plugin.Context) error {
	if len(ctx.Args) < 2 {
		return fmt.Errorf("usage: service remove <service-name> <domain>")
	}

	serviceName := ctx.Args[0]
	domain := ctx.Args[1]

	config, exists := p.configs[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found. Use 'service list' to see available services", serviceName)
	}

	confirm, _ := ctx.Flags["confirm"].(bool)

	if !confirm {
		ctx.Output.Printf("This will remove all %s DNS records from %s.\n", config.DisplayName, domain)
		ctx.Output.Println("Use --confirm to proceed.")
		return nil
	}

	records, err := ctx.DNS.GetRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}

	// Generate expected records to identify what to remove
	expectedRecords := p.generateRecords(config, domain)
	expectedMap := make(map[string]bool)
	for _, record := range expectedRecords {
		key := fmt.Sprintf("%s:%s:%s", record.HostName, record.RecordType, record.Address)
		expectedMap[key] = true
	}

	// Filter out service records
	var filteredRecords []dnsrecord.Record
	removedCount := 0

	for _, record := range records {
		key := fmt.Sprintf("%s:%s:%s", record.HostName, record.RecordType, record.Address)
		if expectedMap[key] {
			removedCount++
			continue
		}

		// Also check by pattern matching for dynamic values
		shouldRemove := false
		for _, expected := range expectedRecords {
			if record.HostName == expected.HostName && record.RecordType == expected.RecordType {
				// An empty needle makes strings.Contains universally true, which
				// would delete every record at this hostname+type. That happens
				// whenever a template's value is exactly the domain (or the
				// domain plus a trailing dot), so guard it explicitly.
				needle := strings.TrimSuffix(strings.TrimSuffix(expected.Address, "."), domain)
				if needle == "" {
					continue
				}
				// Check if address matches pattern
				if strings.Contains(record.Address, needle) {
					shouldRemove = true
					break
				}
			}
		}

		if !shouldRemove {
			filteredRecords = append(filteredRecords, record)
		} else {
			removedCount++
		}
	}

	if removedCount == 0 {
		ctx.Output.Printf("No %s DNS records found for %s\n", config.DisplayName, domain)
		return nil
	}

	err = ctx.DNS.SetRecords(domain, filteredRecords)
	if err != nil {
		return fmt.Errorf("failed to update DNS records: %w", err)
	}

	ctx.Output.Printf("Successfully removed %d %s DNS records from %s\n", removedCount, config.DisplayName, domain)
	return nil
}

// list implements the list command
func (p *ServicePlugin) list(ctx *plugin.Context) error {
	if len(p.configs) == 0 {
		ctx.Output.Println("No service integrations configured.")
		return nil
	}

	ctx.Output.Println("Available Service Integrations:")
	ctx.Output.Println("===============================")

	// Group by category
	categories := make(map[string][]*Config)
	for _, config := range p.configs {
		category := config.Category
		if category == "" {
			category = "other"
		}
		categories[category] = append(categories[category], config)
	}

	for category, configs := range categories {
		ctx.Output.Printf("\n%s:\n", titleCase(category))
		for _, config := range configs {
			ctx.Output.Printf("  %s - %s\n", config.Name, config.DisplayName)
			if config.Description != "" {
				ctx.Output.Printf("    %s\n", config.Description)
			}
		}
	}

	return nil
}

// info implements the info command
func (p *ServicePlugin) info(ctx *plugin.Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: service info <service-name>")
	}

	serviceName := ctx.Args[0]
	config, exists := p.configs[serviceName]
	if !exists {
		return fmt.Errorf("service '%s' not found. Use 'service list' to see available services", serviceName)
	}

	ctx.Output.Printf("Service: %s\n", config.DisplayName)
	ctx.Output.Printf("Name: %s\n", config.Name)
	if config.Description != "" {
		ctx.Output.Printf("Description: %s\n", config.Description)
	}
	if config.Category != "" {
		ctx.Output.Printf("Category: %s\n", config.Category)
	}

	ctx.Output.Println("\nDNS Records:")
	records := p.generateRecords(config, "example.com")
	for _, record := range records {
		mxPref := ""
		if record.MXPref > 0 {
			mxPref = fmt.Sprintf(" (priority: %d)", record.MXPref)
		}
		ctx.Output.Printf("  %s %s → %s%s\n", record.HostName, record.RecordType, record.Address, mxPref)
	}

	return nil
}

// generateRecords generates DNS records from a service integration configuration
func (p *ServicePlugin) generateRecords(config *Config, domainName string) []dnsrecord.Record {
	var records []dnsrecord.Record

	// MX Records
	for _, mx := range config.Records.MX {
		ttl := mx.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}
		records = append(records, dnsrecord.Record{
			HostName:   mx.Hostname,
			RecordType: dnsrecord.RecordTypeMX,
			Address:    ensureTrailingDot(mx.Server),
			TTL:        ttl,
			MXPref:     mx.Priority,
		})
	}

	// SPF Record
	if config.Records.SPF != nil {
		ttl := config.Records.SPF.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}
		records = append(records, dnsrecord.Record{
			HostName:   config.Records.SPF.Hostname,
			RecordType: dnsrecord.RecordTypeTXT,
			Address:    config.Records.SPF.Value,
			TTL:        ttl,
		})
	}

	// DKIM Records
	for _, dkim := range config.Records.DKIM {
		ttl := dkim.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}
		value := dkim.Value
		// Replace {domain} placeholder if present
		value = strings.ReplaceAll(value, "{domain}", domainName)

		recordType := dnsrecord.RecordTypeCNAME
		if dkim.Type == "TXT" {
			recordType = dnsrecord.RecordTypeTXT
		}

		records = append(records, dnsrecord.Record{
			HostName:   dkim.Hostname,
			RecordType: recordType,
			Address:    ensureTrailingDot(value),
			TTL:        ttl,
		})
	}

	// DMARC Record
	if config.Records.DMARC != nil {
		ttl := config.Records.DMARC.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}
		records = append(records, dnsrecord.Record{
			HostName:   config.Records.DMARC.Hostname,
			RecordType: dnsrecord.RecordTypeTXT,
			Address:    config.Records.DMARC.Value,
			TTL:        ttl,
		})
	}

	// Autodiscover
	if config.Records.Autodiscover != nil {
		ttl := config.Records.Autodiscover.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}

		switch config.Records.Autodiscover.Type {
		case "CNAME":
			records = append(records, dnsrecord.Record{
				HostName:   config.Records.Autodiscover.Hostname,
				RecordType: dnsrecord.RecordTypeCNAME,
				Address:    ensureTrailingDot(config.Records.Autodiscover.CNAME),
				TTL:        ttl,
			})
		case "SRV":
			// SRV records are more complex, for now we'll use CNAME
			// Full SRV support can be added later
			if config.Records.Autodiscover.Target != "" {
				records = append(records, dnsrecord.Record{
					HostName:   config.Records.Autodiscover.Hostname,
					RecordType: dnsrecord.RecordTypeCNAME,
					Address:    ensureTrailingDot(config.Records.Autodiscover.Target),
					TTL:        ttl,
				})
			}
		}
	}

	// Custom Records
	for _, custom := range config.Records.Custom {
		ttl := custom.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}
		value := custom.Value
		// Replace {domain} placeholder if present
		value = strings.ReplaceAll(value, "{domain}", domainName)
		value = ensureTrailingDot(value)

		records = append(records, dnsrecord.Record{
			HostName:   custom.Hostname,
			RecordType: custom.Type,
			Address:    value,
			TTL:        ttl,
			MXPref:     custom.MXPref,
		})
	}

	return records
}

// ensureTrailingDot ensures a hostname has a trailing dot if it's a FQDN
func ensureTrailingDot(hostname string) string {
	if hostname == "" {
		return hostname
	}
	// Don't add dot to IP addresses or special values
	if strings.Contains(hostname, " ") || strings.Contains(hostname, "v=") {
		return hostname
	}
	if !strings.HasSuffix(hostname, ".") && strings.Contains(hostname, ".") {
		return hostname + "."
	}
	return hostname
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + strings.ToLower(s[size:])
}
