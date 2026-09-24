# ZoneKit

<div align="center">

![Version](https://img.shields.io/badge/version-0.1.0-blue?style=flat-square)
![Status](https://img.shields.io/badge/status-pre--1.0.0-orange?style=flat-square)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

A command-line interface for managing DNS zones and records across multiple providers with **multi-account support**.

[Installation](#-quick-start) • [Documentation](https://github.com/SamyRai/zonekit/wiki) • [Issues](https://github.com/SamyRai/zonekit/issues) • [Releases](https://github.com/SamyRai/zonekit/releases)

</div>

---

## ⚠️ Disclaimer

> **Warning**
>
> **This is an independent, community-maintained project.**
>
> **Current Status: Pre-1.0.0 Release (v0.1.0)**
>
> This tool is currently in active development and has **not reached version 1.0.0**. As such:
>
> - ⚠️ **Use at your own risk and responsibility**
> - ⚠️ **No warranties or guarantees are provided**
> - ⚠️ **The API may change between versions**
> - ⚠️ **Always test in a sandbox environment first**
> - ⚠️ **Backup your DNS records before making bulk changes**
> - ⚠️ **Review changes carefully before applying them**
>
> The maintainers are not responsible for any data loss, service disruption, or other issues that may arise from using this tool. Please report bugs and contribute improvements via GitHub issues and pull requests.
>
> For version information and release notes, see [VERSIONING.md](VERSIONING.md).

## Features

| Feature | Description |
|---------|-------------|
| **Multi-Provider Support** | Support for multiple DNS providers (Namecheap, Cloudflare, and more) |
| **Multi-Account Management** | Configure and switch between multiple provider accounts |
| **Domain Management** | List, check, and manage your domains |
| **DNS Management** | Create, update, and delete DNS records |
| **Bulk Operations** | Perform multiple DNS operations at once |
| **Account Switching** | Easy switching between different accounts |
| **Plugin System** | Extensible plugin architecture for custom functionality |
| **Secure Configuration** | API keys and credentials stored securely |

## Quick Start

<details>
<summary><strong>Click to expand quick start guide</strong></summary>

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/SamyRai/zonekit.git
cd namecheap

# Build the binary
make build

# Or build directly
go build -o zonekit ./main.go
```

### 2. Configuration

The tool automatically detects configuration files in this priority order:

| Priority | Location | Use Case |
|----------|----------|----------|
| **1** | `./configs/.zonekit.yaml` | Development |
| **2** | `~/.zonekit.yaml` | Production |

```bash
# Initialize configuration
./zonekit config init

# Or add account interactively
./zonekit account add
```

### 3. Test Your Setup

```bash
# List accounts
./zonekit account list

# List domains
./zonekit domain list

# Use specific account
./zonekit --account work domain list
```

</details>

> **For detailed documentation, see the [Wiki](https://github.com/SamyRai/zonekit/wiki)**

## Commands

<details>
<summary><strong>Global flags</strong></summary>

| Flag | Description |
|------|-------------|
| `--output`, `-o` | Output format: `table` (default, human-readable) \| `json` \| `yaml`. Every list/read command below emits stable, snake_case fields in json/yaml mode. Diagnostics (`Using account: ...`, `Using config file: ...`) always go to stderr, never stdout, in every mode, so stdout stays safe to pipe or parse. On error, table mode prints `Error: <message>` and json/yaml mode prints `{"error": "<message>"}` (or the YAML equivalent) — both to stderr, both with a non-zero exit code. |
| `--account` | Use a specific account instead of the current one |
| `--config` | Use a specific config file |

</details>

<details>
<summary><strong>Account Management</strong></summary>

| Command | Description |
|---------|-------------|
| `account list` | List all accounts |
| `account add [name]` | Add new account |
| `account switch <name>` | Switch to account |
| `account show [name]` | Show account details |
| `account edit [name]` | Edit account |
| `account remove <name>` | Remove account |

</details>

<details>
<summary><strong>Domain Management</strong></summary>

| Command | Description |
|---------|-------------|
| `domain list` | List all domains (`--output json\|yaml` supported) |
| `domain info <domain>` | Get domain details (`--output json\|yaml` supported) |
| `domain check <domain>` | Check availability |
| `domain renew <domain> [years]` | Renew domain |
| `domain nameservers get <domain>` | Get nameservers (`--output json\|yaml` supported) |
| `domain nameservers set <domain> <ns1> [ns2]... [--dry-run]` | Set nameservers |
| `domain nameservers default <domain> [--dry-run]` | Reset to default |

</details>

<details>
<summary><strong>DNS Management</strong></summary>

| Command | Description |
|---------|-------------|
| `dns list <domain> [--type T] [--name N]` | List DNS records. `--name` is an exact, case-insensitive hostname match (not a substring match) |
| `dns add <domain> <host> <type> <value> [--ttl] [--mx-pref] [--dry-run]` | Add DNS record |
| `dns update <domain> <host> <type> <value> [--match-value] [--dry-run]` | Update DNS record |
| `dns delete <domain> <host> <type> [--dry-run]` | Delete DNS record |
| `dns clear <domain> --confirm [--dry-run]` | Clear all records. Refuses without `--confirm`; `--dry-run` previews without needing `--confirm` |
| `dns bulk <domain> <file> --confirm [--dry-run]` | Bulk operations |
| `dns ensure <domain> <host> <type> <value> [--ttl] [--mx-pref] [--dry-run]` | Idempotent create-or-update: reports `created`, `updated`, or `unchanged`. See below for the identity rule |
| `dns import <domain> <file> --confirm [--dry-run]` | Import zone file (replaces ALL records) |
| `dns export <domain> [file]` | Export zone file |

`--dry-run` is supported on every mutating command above (`add`, `update`, `delete`, `clear`, `bulk`, `ensure`, `domain nameservers set`/`default`) and makes **no API writes**: it reports what would change (respecting `--output`) and exits without calling the provider.

**`dns ensure` identity rule:** for `MX` and `TXT` records, `(hostname, type, value)` identifies the record — a hostname can legitimately carry several of each at once (multiple MX priorities; SPF/DMARC/verification TXT records), so a value that doesn't match an existing record is added alongside the others rather than overwriting one of them. For every other type, `(hostname, type)` identifies the record, matching normal DNS practice of one A/AAAA/CNAME/NS per hostname; if more than one already exists, `ensure` refuses rather than guess which to update.

**JSON output schema** (stable, snake_case; same shape in `--output yaml`):

```jsonc
// dns list --output json
[
  { "hostname": "@", "type": "MX", "value": "mail.example.com", "ttl": 1800, "mx_pref": 10 },
  { "hostname": "www", "type": "A", "value": "192.168.1.1" }
]

// dns ensure --output json
{ "status": "created", "record": { "hostname": "www", "type": "A", "value": "192.168.1.1", "ttl": 1800 } }
```

`id` is included when the provider reports one (`omitempty` otherwise). `ttl`/`mx_pref` are omitted when zero.

**Zone file format notes** (`dns export`/`dns import`, `pkg/dns/zonefile`):

- Every record line always carries an explicit owner (`@` for the zone apex, or the
  relative hostname). BIND master-file syntax lets a blank owner inherit the *previous*
  line's owner, so a blank owner is never emitted -- it would silently relocate a record
  onto whatever hostname preceded it.
- Namecheap's `URL`/`URL301`/`FRAME` entries configure the registrar's own HTTP
  redirect/frame forwarding; they are not DNS resource records. They are exported as
  inert `; zonekit:url-redirect <owner> <type> "<value>"` comments, not as `IN URL...`
  lines, and `dns import` restores them from that comment form.
- `SOA`/`NS` are intentionally **not** exported. Namecheap's hosted-DNS API doesn't expose
  real SOA data (serial/refresh/retry/expire) and manages the zone's NS set itself, so
  there is nothing accurate to emit; a placeholder SOA/NS block would only be fabricated
  data. The exported file documents this omission in its header comment.
- Long `TXT` values (e.g. a 408-character DKIM key) are split into multiple quoted
  `<character-string>`s of at most 255 bytes each, per RFC 1035; `dns import` reassembles
  them into the original value.
- `dns import` replaces **all** existing records for the domain (it lists what it parsed
  and requires `--confirm` to apply).

</details>

> **For complete command reference, see [Usage Guide](https://github.com/SamyRai/zonekit/wiki/Usage)**

## Security

- Configuration files use `600` permissions (owner read/write only)
- API keys are masked in output
- Configuration files are excluded from git by default
- Sensitive data is encrypted in memory

## Configuration File Locations

The tool automatically detects configuration files in this priority order:

1. **Project Directory** (Recommended for development):
   - `./configs/.zonekit.yaml`
   - Automatically found when running from project directory

2. **Home Directory** (Fallback):
   - `~/.zonekit.yaml`
   - Used when no project config is found

3. **Custom Location**:
   - `./zonekit --config /path/to/config.yaml`

## Pro Tips

### Multi-Account Workflow

```bash
# 1. Add multiple accounts
./zonekit account add personal
./zonekit account add work
./zonekit account add client1

# 2. Switch between accounts
./zonekit account switch work
./zonekit domain list

./zonekit account switch personal
./zonekit domain list

# 3. Use specific account for one-off commands
./zonekit --account work dns list example.com
./zonekit --account personal domain check newdomain.com
```

### Account Organization

- Use descriptive names: `personal`, `work`, `client1`, `client2`
- Add descriptions for better organization
- Keep related domains in the same account
- Use sandbox accounts for testing

## Troubleshooting

### Common Issues

1. **"No config file found"**
   - Run `./zonekit config init` to create a config file
   - Ensure the config file is in the correct location

2. **"Account not found"**
   - Check available accounts with `./zonekit account list`
   - Verify account names are correct

3. **API Connection Errors**
   - Verify your API key is correct
   - Check that your client IP is correct
   - Ensure you're not using sandbox credentials in production

### Getting Help

```bash
# Help
./zonekit help

# Command-specific help
./zonekit account --help
./zonekit domain --help
./zonekit dns --help
```

## Migration from Legacy Config

If you have an existing single-account configuration, the tool will automatically migrate it to the new multi-account format. Your existing configuration will be preserved as the `default` account.

## Development

### Project Structure

```
namecheap/
├── cmd/                    # Command implementations
├── pkg/                    # Core packages
│   ├── client/            # Namecheap API client
│   ├── config/            # Configuration management
│   ├── domain/            # Domain operations
│   └── dns/               # DNS operations
├── configs/                # Configuration files
├── internal/               # Internal packages
└── main.go                 # Entry point
```

### Building

```bash
# Development build
go build -o zonekit cmd/main.go

# Production build
make build

# Install to system
make install
```

## Using as a Library

ZoneKit's module path is `go.glpx.pro/zonekit`, served by a Go vanity import
server in front of this private Gitea repository. The packages under `pkg/`
(for example `pkg/dns/provider`, `pkg/dns`, `pkg/client`, `pkg/config`,
`pkg/domain`, `pkg/dnsrecord`, `pkg/validation`) are the public library
surface and are safe to import directly; only packages under `internal/`
(CLI-only helpers such as `internal/cmdutil`) are off-limits to importers.

Because the repository behind `go.glpx.pro/zonekit` is private, `go get`
resolves it over SSH via Gitea rather than a public checksum database.
**Set `GOPRIVATE` before fetching it** so the Go toolchain skips the public
module proxy and sum database for this host:

```bash
export GOPRIVATE=go.glpx.pro
go get go.glpx.pro/zonekit/pkg/dns/provider
```

Then import it like any other module:

```go
import "go.glpx.pro/zonekit/pkg/dns/provider"
```

Fetching requires SSH access to `gitea.bk.glpx.pro` (the vanity server only
serves the module-discovery metadata; `git`/`go` still clone the repository
directly). See `docs/glpxctl.md` in the `cluster` repo for how CI consumers
are granted read access.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions:
- Check the troubleshooting section above
- Review the help command: `./zonekit help`
- Open an issue on GitHub
