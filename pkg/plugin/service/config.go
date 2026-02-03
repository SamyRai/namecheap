package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents a service integration configuration
type Config struct {
	Name         string        `yaml:"name"`
	DisplayName  string        `yaml:"display_name"`
	Description  string        `yaml:"description"`
	Category     string        `yaml:"category"` // email, cdn, hosting, etc.
	Records      Records       `yaml:"records"`
	Verification *Verification `yaml:"verification,omitempty"`
}

// Records defines all DNS records for a service integration
type Records struct {
	MX           []MXRecord          `yaml:"mx,omitempty"`
	SPF          *TXTRecord          `yaml:"spf,omitempty"`
	DKIM         []DKIMRecord        `yaml:"dkim,omitempty"`
	DMARC        *TXTRecord          `yaml:"dmarc,omitempty"`
	Autodiscover *AutodiscoverRecord `yaml:"autodiscover,omitempty"`
	Custom       []CustomRecord      `yaml:"custom,omitempty"`
}

// MXRecord represents an MX record
type MXRecord struct {
	Hostname string `yaml:"hostname"`
	Server   string `yaml:"server"`
	Priority int    `yaml:"priority"`
	TTL      int    `yaml:"ttl,omitempty"`
}

// TXTRecord represents a TXT record
type TXTRecord struct {
	Hostname string `yaml:"hostname"`
	Value    string `yaml:"value"`
	TTL      int    `yaml:"ttl,omitempty"`
}

// DKIMRecord represents a DKIM record (can be CNAME or TXT)
type DKIMRecord struct {
	Hostname string `yaml:"hostname"`
	Type     string `yaml:"type"` // CNAME or TXT
	Value    string `yaml:"value"`
	TTL      int    `yaml:"ttl,omitempty"`
}

// AutodiscoverRecord represents autodiscover configuration
type AutodiscoverRecord struct {
	Type     string `yaml:"type"` // SRV or CNAME
	Hostname string `yaml:"hostname"`
	// For SRV
	Service  string `yaml:"service,omitempty"`
	Target   string `yaml:"target,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Priority int    `yaml:"priority,omitempty"`
	Weight   int    `yaml:"weight,omitempty"`
	// For CNAME
	CNAME string `yaml:"cname,omitempty"`
	TTL   int    `yaml:"ttl,omitempty"`
}

// CustomRecord represents a custom DNS record
type CustomRecord struct {
	Hostname string `yaml:"hostname"`
	Type     string `yaml:"type"` // A, AAAA, CNAME, TXT, NS, SRV
	Value    string `yaml:"value"`
	TTL      int    `yaml:"ttl,omitempty"`
	MXPref   int    `yaml:"mx_pref,omitempty"`
}

// Verification defines how to verify provider setup
type Verification struct {
	RequiredRecords []VerificationCheck `yaml:"required_records,omitempty"`
}

// VerificationCheck defines a single verification check
type VerificationCheck struct {
	Type       string `yaml:"type"`
	Hostname   string `yaml:"hostname"`
	Contains   string `yaml:"contains,omitempty"`
	Equals     string `yaml:"equals,omitempty"`
	StartsWith string `yaml:"starts_with,omitempty"`
}

// LoadConfig loads a service integration configuration from a YAML file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// LoadAllConfigs loads all service integration configurations from a directory
func LoadAllConfigs(dirPath string) (map[string]*Config, error) {
	configs := make(map[string]*Config)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		config, err := LoadConfig(filePath)
		if err != nil {
			// Log error but continue loading other configs
			continue
		}

		configs[config.Name] = config
	}

	return configs, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}

	// Validate MX records
	for i, mx := range c.Records.MX {
		if mx.Hostname == "" {
			return fmt.Errorf("mx[%d].hostname is required", i)
		}
		if mx.Server == "" {
			return fmt.Errorf("mx[%d].server is required", i)
		}
		if mx.Priority < 0 {
			return fmt.Errorf("mx[%d].priority must be non-negative", i)
		}
	}

	// Validate SPF
	if c.Records.SPF != nil {
		if c.Records.SPF.Hostname == "" {
			return fmt.Errorf("spf.hostname is required")
		}
		if c.Records.SPF.Value == "" {
			return fmt.Errorf("spf.value is required")
		}
	}

	// Validate DKIM records
	for i, dkim := range c.Records.DKIM {
		if dkim.Hostname == "" {
			return fmt.Errorf("dkim[%d].hostname is required", i)
		}
		if dkim.Type != "CNAME" && dkim.Type != "TXT" {
			return fmt.Errorf("dkim[%d].type must be CNAME or TXT", i)
		}
		if dkim.Value == "" {
			return fmt.Errorf("dkim[%d].value is required", i)
		}
	}

	// Validate DMARC
	if c.Records.DMARC != nil {
		if c.Records.DMARC.Hostname == "" {
			return fmt.Errorf("dmarc.hostname is required")
		}
		if c.Records.DMARC.Value == "" {
			return fmt.Errorf("dmarc.value is required")
		}
	}

	// Validate custom records
	for i, custom := range c.Records.Custom {
		if custom.Hostname == "" {
			return fmt.Errorf("custom[%d].hostname is required", i)
		}
		if custom.Type == "" {
			return fmt.Errorf("custom[%d].type is required", i)
		}
		if custom.Value == "" {
			return fmt.Errorf("custom[%d].value is required", i)
		}
	}

	return nil
}
