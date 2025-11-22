package validation

import (
	"fmt"
	"strings"
)

// ValidateDomain validates a domain name format.
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Basic domain validation
	if len(domain) > 253 {
		return fmt.Errorf("domain name too long (max 253 characters)")
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid domain format: must contain at least one dot")
	}

	for _, part := range parts {
		if len(part) == 0 {
			return fmt.Errorf("invalid domain format: empty label")
		}
		if len(part) > 63 {
			return fmt.Errorf("invalid domain format: label too long (max 63 characters)")
		}
	}

	return nil
}

