# Namecheap DNS Manager

A comprehensive command-line interface for managing Namecheap domains and DNS records with **multi-account support**.

## 🎯 Features

- **Multi-Account Management**: Configure and switch between multiple Namecheap accounts
- **Domain Management**: List, check, and manage your domains
- **DNS Management**: Create, update, and delete DNS records
- **Bulk Operations**: Perform multiple DNS operations at once
- **Account Switching**: Easy switching between different accounts
- **Secure Configuration**: API keys and credentials stored securely

## 🚀 Quick Start

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/SamyRai/namecheap.git
cd namecheap

# Build the binary
go build -o namecheap-dns cmd/main.go

# Or use make
make build
```

### 2. Configuration

The tool automatically looks for configuration files in this order:
1. **Project directory**: `./configs/.namecheap-dns.yaml` (recommended for development)
2. **Home directory**: `~/.namecheap-dns.yaml` (fallback)

#### Initialize Configuration

```bash
# Create a new configuration file
./namecheap-dns config init

# Or manually create configs/.namecheap-dns.yaml
```

#### Example Configuration

```yaml
# configs/.namecheap-dns.yaml
accounts:
  default:
    username: "your-namecheap-username"
    api_user: "your-api-username"
    api_key: "your-api-key-here"
    client_ip: "your.public.ip.address"
    use_sandbox: false
    description: "My main account"
  
  work:
    username: "work-username"
    api_user: "work-api-username"
    api_key: "work-api-key"
    client_ip: "your.public.ip.address"
    use_sandbox: false
    description: "Work account"

current_account: "default"
```

### 3. Add Your First Account

```bash
# Add an account interactively
./namecheap-dns account add

# Or add with a specific name
./namecheap-dns account add personal
```

### 4. Test Your Setup

```bash
# List all accounts
./namecheap-dns account list

# Show current account
./namecheap-dns account show

# List domains (using current account)
./namecheap-dns domain list

# Use a specific account for a command
./namecheap-dns --account work domain list
```

## 📋 Commands

### Account Management

```bash
./namecheap-dns account list                    # List all accounts
./namecheap-dns account add [name]              # Add new account
./namecheap-dns account switch <name>           # Switch to account
./namecheap-dns account show [name]             # Show account details
./namecheap-dns account edit [name]             # Edit account
./namecheap-dns account remove <name>           # Remove account
```

### Domain Management

```bash
./namecheap-dns domain list                     # List all domains
./namecheap-dns domain info <domain>            # Get domain details
./namecheap-dns domain check <domain>           # Check availability
./namecheap-dns domain renew <domain> [years]   # Renew domain
./namecheap-dns domain nameservers get <domain> # Get nameservers
./namecheap-dns domain nameservers set <domain> <ns1> [ns2] [ns3] [ns4]
./namecheap-dns domain nameservers default <domain>
```

### DNS Management

```bash
./namecheap-dns dns list <domain>               # List DNS records
./namecheap-dns dns add <domain> <host> <type> <value>
./namecheap-dns dns update <domain> <host> <type> <value>
./namecheap-dns dns delete <domain> <host> <type>
./namecheap-dns dns clear <domain>              # Clear all records
./namecheap-dns dns bulk <domain> <file>        # Bulk operations
./namecheap-dns dns import <domain> <file>      # Import zone file
./namecheap-dns dns export <domain> [file]      # Export zone file
```

### Configuration (Legacy)

```bash
./namecheap-dns config init                      # Initialize config
./namecheap-dns config set                       # Set config interactively
./namecheap-dns config show                      # Show current config
./namecheap-dns config validate                  # Validate config
```

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
# Comprehensive help
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
