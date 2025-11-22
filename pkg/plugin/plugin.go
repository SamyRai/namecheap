package plugin

import (
	"namecheap-dns-manager/pkg/dns"
)

// Service defines the DNS service interface for plugins
type Service interface {
	GetRecords(domainName string) ([]dns.Record, error)
	GetRecordsByType(domainName string, recordType string) ([]dns.Record, error)
	SetRecords(domainName string, records []dns.Record) error
	AddRecord(domainName string, record dns.Record) error
	UpdateRecord(domainName string, hostname, recordType string, newRecord dns.Record) error
	DeleteRecord(domainName string, hostname, recordType string) error
	DeleteAllRecords(domainName string) error
	ValidateRecord(record dns.Record) error
	BulkUpdate(domainName string, operations []dns.BulkOperation) error
}

// Plugin defines the interface that all plugins must implement.
type Plugin interface {
	// Name returns the unique name of the plugin
	Name() string

	// Description returns a human-readable description of the plugin
	Description() string

	// Version returns the plugin version
	Version() string

	// Commands returns the list of commands this plugin provides
	Commands() []Command
}

// CommandFunc is the function type for executing a plugin command
type CommandFunc func(ctx *Context) error

// Command represents a command that a plugin can execute
type Command struct {
	// Name is the command name (e.g., "setup", "verify", "remove")
	Name string

	// Description is a short description of what the command does
	Description string

	// LongDescription is a detailed description
	LongDescription string

	// Execute runs the command with the given context
	Execute CommandFunc
}

// Context provides the execution context for plugin commands
type Context struct {
	// Domain is the domain name being operated on
	Domain string

	// DNS is the DNS service for managing records
	DNS Service

	// Args are additional command arguments
	Args []string

	// Flags are command flags
	Flags map[string]interface{}

	// Output is for writing output messages
	Output OutputWriter
}

// OutputWriter provides a way for plugins to write output
type OutputWriter interface {
	Printf(format string, args ...interface{})
	Println(args ...interface{})
	Print(args ...interface{})
}

// SetupResult represents the result of a setup operation
type SetupResult struct {
	Records   []dns.Record
	Conflicts []Conflict
	NextSteps []string
	DryRun    bool
	Replace   bool
}

// Conflict represents a conflict with existing DNS records
type Conflict struct {
	HostName   string
	RecordType string
	Existing   dns.Record
	New        dns.Record
}

// VerificationResult represents the result of a verification operation
type VerificationResult struct {
	Valid   bool
	Checks  []VerificationCheck
	Message string
}

// VerificationCheck represents a single verification check
type VerificationCheck struct {
	Name    string
	Status  bool // true = passed, false = failed
	Message string
}
