# Plugin System

The plugin system allows you to extend the functionality of namecheap-dns-manager with custom integrations.

## Architecture

### Plugin Interface

All plugins must implement the `plugin.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Description() string
    Version() string
    Commands() []Command
}
```

### Plugin Commands

Each plugin can provide multiple commands. Commands receive a `Context` that provides:
- Domain name
- DNS service for managing records
- Command arguments and flags
- Output writer for displaying results

## Creating a Plugin

### Example: Email Provider Plugin

```go
package myemail

import (
    "namecheap-dns-manager/pkg/dns"
    "namecheap-dns-manager/pkg/plugin"
)

type MyEmailPlugin struct{}

func New() *MyEmailPlugin {
    return &MyEmailPlugin{}
}

func (p *MyEmailPlugin) Name() string {
    return "myemail"
}

func (p *MyEmailPlugin) Description() string {
    return "My Email Provider integration"
}

func (p *MyEmailPlugin) Version() string {
    return "1.0.0"
}

func (p *MyEmailPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        {
            Name:        "setup",
            Description: "Set up email DNS records",
            Execute:     p.setup,
        },
    }
}

func (p *MyEmailPlugin) setup(ctx *plugin.Context) error {
    records := []dns.Record{
        {
            HostName:   "@",
            RecordType: dns.RecordTypeMX,
            Address:    "mail.example.com.",
            TTL:        dns.DefaultTTL,
            MXPref:     10,
        },
    }

    ctx.Output.Printf("Setting up email records for %s\n", ctx.Domain)
    return ctx.DNS.SetRecords(ctx.Domain, records)
}
```

### Registering Your Plugin

In `cmd/root.go`, add your plugin registration:

```go
func initPlugins() {
    // Existing plugins...
    plugin.Register(migadu.New())

    // Your plugin
    plugin.Register(myemail.New())
}
```

## Using Plugins

### List Available Plugins

```bash
namecheap-dns plugin list
```

### Get Plugin Information

```bash
namecheap-dns plugin info migadu
```

### Execute Plugin Command

```bash
namecheap-dns plugin migadu setup example.com
namecheap-dns plugin migadu verify example.com --dry-run
namecheap-dns plugin migadu remove example.com --confirm
```

## Built-in Plugins

### Migadu

Email hosting provider integration with commands:
- `setup` - Configure DNS records for Migadu
- `verify` - Verify Migadu DNS configuration
- `remove` - Remove Migadu DNS records

## Plugin Context

The `Context` provided to plugin commands includes:

- `Domain` - The domain name being operated on
- `DNS` - DNS service instance for managing records
- `Args` - Additional command arguments
- `Flags` - Command flags (dry-run, replace, confirm, etc.)
- `Output` - Output writer for displaying messages

## Best Practices

1. **Validate Input**: Always validate domain names and other inputs
2. **Dry Run Support**: Support `--dry-run` flag to preview changes
3. **Conflict Detection**: Check for existing records before modifying
4. **Clear Output**: Use the Output writer for all user-facing messages
5. **Error Handling**: Return descriptive errors with context
6. **Documentation**: Provide clear descriptions for all commands

## Extending the System

To add new plugin types or capabilities:

1. Extend the `Plugin` interface if needed
2. Add new fields to `Context` for additional capabilities
3. Update the plugin registry to support new features
4. Document the changes in this README

