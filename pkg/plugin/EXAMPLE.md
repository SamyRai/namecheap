# Plugin Example: Creating a Custom Email Provider Plugin

This example shows how to create a plugin for a different email provider (e.g., Google Workspace, Microsoft 365, etc.).

## Step 1: Create Plugin Package

Create a new directory for your plugin:

```bash
mkdir -p pkg/plugin/myemail
```

## Step 2: Implement Plugin Interface

Create `pkg/plugin/myemail/myemail.go`:

```go
package myemail

import (
    "fmt"
    "namecheap-dns-manager/pkg/dns"
    "namecheap-dns-manager/pkg/plugin"
)

const (
    pluginName        = "myemail"
    pluginVersion     = "1.0.0"
    pluginDescription = "My Email Provider integration"
)

type MyEmailPlugin struct{}

func New() *MyEmailPlugin {
    return &MyEmailPlugin{}
}

func (p *MyEmailPlugin) Name() string {
    return pluginName
}

func (p *MyEmailPlugin) Description() string {
    return pluginDescription
}

func (p *MyEmailPlugin) Version() string {
    return pluginVersion
}

func (p *MyEmailPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {
            Name:        "setup",
            Description: "Set up email DNS records",
            LongDescription: `Set up all necessary DNS records for My Email Provider.
This will add:
- MX records for mail routing
- SPF record for sender authentication
- DKIM records for email signing
- DMARC record for email policy`,
            Execute: func(ctx *plugin.Context) error {
                return p.setup(ctx)
            },
        },
        {
            Name:        "verify",
            Description: "Verify email DNS records",
            Execute: func(ctx *plugin.Context) error {
                return p.verify(ctx)
            },
        },
    }
}

func (p *MyEmailPlugin) setup(ctx *plugin.Context) error {
    dryRun, _ := ctx.Flags["dry-run"].(bool)
    replace, _ := ctx.Flags["replace"].(bool)

    // Get existing records if not replacing
    var existingRecords []dns.Record
    var err error
    if !replace {
        existingRecords, err = ctx.DNS.GetRecords(ctx.Domain)
        if err != nil {
            return fmt.Errorf("failed to get existing records: %w", err)
        }
    }

    // Generate DNS records for your email provider
    records := []dns.Record{
        {
            HostName:   "@",
            RecordType: dns.RecordTypeMX,
            Address:    "mail.example.com.",
            TTL:        dns.DefaultTTL,
            MXPref:     10,
        },
        {
            HostName:   "@",
            RecordType: dns.RecordTypeTXT,
            Address:    "v=spf1 include:_spf.example.com -all",
            TTL:        dns.DefaultTTL,
        },
    }

    ctx.Output.Printf("Setting up email DNS records for %s\n", ctx.Domain)

    if dryRun {
        ctx.Output.Println("DRY RUN MODE - No changes will be made")
        ctx.Output.Println("Records to be added:")
        for _, record := range records {
            ctx.Output.Printf("  %s %s → %s\n", record.HostName, record.RecordType, record.Address)
        }
        return nil
    }

    // Apply changes
    var allRecords []dns.Record
    if replace {
        allRecords = records
    } else {
        allRecords = existingRecords
        allRecords = append(allRecords, records...)
    }

    err = ctx.DNS.SetRecords(ctx.Domain, allRecords)
    if err != nil {
        return fmt.Errorf("failed to set DNS records: %w", err)
    }

    ctx.Output.Printf("✅ Successfully set up email DNS records for %s\n", ctx.Domain)
    return nil
}

func (p *MyEmailPlugin) verify(ctx *plugin.Context) error {
    records, err := ctx.DNS.GetRecords(ctx.Domain)
    if err != nil {
        return fmt.Errorf("failed to get DNS records: %w", err)
    }

    ctx.Output.Printf("Verifying email setup for %s\n", ctx.Domain)

    // Check for required records
    hasMX := false
    hasSPF := false

    for _, record := range records {
        if record.RecordType == dns.RecordTypeMX && record.HostName == "@" {
            hasMX = true
        }
        if record.RecordType == dns.RecordTypeTXT &&
           record.HostName == "@" &&
           strings.Contains(record.Address, "v=spf1") {
            hasSPF = true
        }
    }

    if hasMX {
        ctx.Output.Println("✅ MX record found")
    } else {
        ctx.Output.Println("❌ MX record missing")
    }

    if hasSPF {
        ctx.Output.Println("✅ SPF record found")
    } else {
        ctx.Output.Println("❌ SPF record missing")
    }

    return nil
}
```

## Step 3: Register Plugin

In `cmd/root.go`, add your plugin registration:

```go
func initPlugins() {
    // Existing plugins
    plugin.Register(migadu.New())

    // Your plugin
    plugin.Register(myemail.New())
}
```

Don't forget to import:

```go
import (
    "namecheap-dns-manager/pkg/plugin/myemail"
)
```

## Step 4: Use Your Plugin

```bash
# List plugins
namecheap-dns plugin list

# Get plugin info
namecheap-dns plugin info myemail

# Setup email
namecheap-dns plugin myemail setup example.com

# Verify setup
namecheap-dns plugin myemail verify example.com

# Dry run
namecheap-dns plugin myemail setup example.com --dry-run
```

## Practices

1. **Validate Input**: Always validate domain names and inputs
2. **Support Dry Run**: Implement `--dry-run` flag support
3. **Conflict Detection**: Check for existing records before modifying
4. **Clear Output**: Use `ctx.Output` for all user-facing messages
5. **Error Handling**: Return descriptive errors with context
6. **Documentation**: Provide clear descriptions for all commands
7. **Use Constants**: Use DNS constants from `dns` package (e.g., `dns.RecordTypeMX`, `dns.DefaultTTL`)

