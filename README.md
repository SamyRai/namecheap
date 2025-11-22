# Namecheap DNS Manager

<div align="center">

![Version](https://img.shields.io/badge/version-0.1.0-blue?style=flat-square)
![Status](https://img.shields.io/badge/status-pre--1.0.0-orange?style=flat-square)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

A command-line interface for managing Namecheap domains and DNS records with **multi-account support**.

[Installation](#-quick-start) • [Documentation](https://github.com/SamyRai/namecheap/wiki) • [Issues](https://github.com/SamyRai/namecheap/issues) • [Releases](https://github.com/SamyRai/namecheap/releases)

</div>

---

## ⚠️ Disclaimer

> **Warning**
> 
> **This is NOT an official Namecheap tool.** This is an independent, community-maintained project.
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

## 🎯 Features

| Feature | Description |
|---------|-------------|
| 🔐 **Multi-Account Management** | Configure and switch between multiple Namecheap accounts |
| 🌐 **Domain Management** | List, check, and manage your domains |
| 📝 **DNS Management** | Create, update, and delete DNS records |
| ⚡ **Bulk Operations** | Perform multiple DNS operations at once |
| 🔄 **Account Switching** | Easy switching between different accounts |
| 🔌 **Plugin System** | Extensible plugin architecture for custom functionality |
| 🔒 **Secure Configuration** | API keys and credentials stored securely |

## 🚀 Quick Start

<details>
<summary><strong>Click to expand quick start guide</strong></summary>

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/SamyRai/namecheap.git
cd namecheap

# Build the binary
make build

# Or build directly
go build -o namecheap-dns ./main.go
```

### 2. Configuration

The tool automatically detects configuration files in this priority order:

| Priority | Location | Use Case |
|----------|----------|----------|
| **1** | `./configs/.namecheap-dns.yaml` | 🛠️ Development |
| **2** | `~/.namecheap-dns.yaml` | 🏠 Production |

```bash
# Initialize configuration
./namecheap-dns config init

# Or add account interactively
./namecheap-dns account add
```

### 3. Test Your Setup

```bash
# List accounts
./namecheap-dns account list

# List domains
./namecheap-dns domain list

# Use specific account
./namecheap-dns --account work domain list
```

</details>

> **📚 For detailed documentation, see the [Wiki](https://github.com/SamyRai/namecheap/wiki)**

## 📋 Commands

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
| `domain list` | List all domains |
| `domain info <domain>` | Get domain details |
| `domain check <domain>` | Check availability |
| `domain renew <domain> [years]` | Renew domain |
| `domain nameservers get <domain>` | Get nameservers |
| `domain nameservers set <domain> <ns1> [ns2]...` | Set nameservers |
| `domain nameservers default <domain>` | Reset to default |

</details>

<details>
<summary><strong>DNS Management</strong></summary>

| Command | Description |
|---------|-------------|
| `dns list <domain>` | List DNS records |
| `dns add <domain> <host> <type> <value>` | Add DNS record |
| `dns update <domain> <host> <type> <value>` | Update DNS record |
| `dns delete <domain> <host> <type>` | Delete DNS record |
| `dns clear <domain>` | Clear all records |
| `dns bulk <domain> <file>` | Bulk operations |
| `dns import <domain> <file>` | Import zone file |
| `dns export <domain> [file]` | Export zone file |

</details>

> **📖 For complete command reference, see [Usage Guide](https://github.com/SamyRai/namecheap/wiki/Usage)**

## 🔐 Security

- Configuration files use `600` permissions (owner read/write only)
- API keys are masked in output
- Configuration files are excluded from git by default
- Sensitive data is encrypted in memory

## 📁 Configuration File Locations

The tool automatically detects configuration files in this priority order:

1. **Project Directory** (Recommended for development):
   - `./configs/.namecheap-dns.yaml`
   - Automatically found when running from project directory

2. **Home Directory** (Fallback):
   - `~/.namecheap-dns.yaml`
   - Used when no project config is found

3. **Custom Location**:
   - `./namecheap-dns --config /path/to/config.yaml`

## 💡 Pro Tips

### Multi-Account Workflow

```bash
# 1. Add multiple accounts
./namecheap-dns account add personal
./namecheap-dns account add work
./namecheap-dns account add client1

# 2. Switch between accounts
./namecheap-dns account switch work
./namecheap-dns domain list

./namecheap-dns account switch personal
./namecheap-dns domain list

# 3. Use specific account for one-off commands
./namecheap-dns --account work dns list example.com
./namecheap-dns --account personal domain check newdomain.com
```

### Account Organization

- Use descriptive names: `personal`, `work`, `client1`, `client2`
- Add descriptions for better organization
- Keep related domains in the same account
- Use sandbox accounts for testing

## 🆘 Troubleshooting

### Common Issues

1. **"No config file found"**
   - Run `./namecheap-dns config init` to create a config file
   - Ensure the config file is in the correct location

2. **"Account not found"**
   - Check available accounts with `./namecheap-dns account list`
   - Verify account names are correct

3. **API Connection Errors**
   - Verify your API key is correct
   - Check that your client IP is correct
   - Ensure you're not using sandbox credentials in production

### Getting Help

```bash
# Help
./namecheap-dns help

# Command-specific help
./namecheap-dns account --help
./namecheap-dns domain --help
./namecheap-dns dns --help
```

## 🔄 Migration from Legacy Config

If you have an existing single-account configuration, the tool will automatically migrate it to the new multi-account format. Your existing configuration will be preserved as the `default` account.

## 📝 Development

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
go build -o namecheap-dns cmd/main.go

# Production build
make build

# Install to system
make install
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📞 Support

For issues and questions:
- Check the troubleshooting section above
- Review the help command: `./namecheap-dns help`
- Open an issue on GitHub
