package dns

import (
	"fmt"
	"net"
	"strings"

	"namecheap-dns-manager/pkg/validation"
)

// ValidateDomain validates a domain name format.
func ValidateDomain(domain string) error {
	return validation.ValidateDomain(domain)
}

// ValidateHostname validates a hostname format.
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	// Allow @ for root domain
	if hostname == "@" {
		return nil
	}

	// Basic hostname validation
	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long (max 253 characters)")
	}

	parts := strings.Split(hostname, ".")
	for _, part := range parts {
		if len(part) == 0 {
			return fmt.Errorf("invalid hostname format: empty label")
		}
		if len(part) > 63 {
			return fmt.Errorf("invalid hostname format: label too long (max 63 characters)")
		}
	}

	return nil
}

// ValidateIPv4 validates an IPv4 address.
func ValidateIPv4(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IPv4 address: %s", ip)
	}
	if parsed.To4() == nil {
		return fmt.Errorf("not an IPv4 address: %s", ip)
	}
	return nil
}

// ValidateIPv6 validates an IPv6 address.
func ValidateIPv6(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IPv6 address: %s", ip)
	}
	if parsed.To4() != nil {
		return fmt.Errorf("not an IPv6 address: %s", ip)
	}
	return nil
}
