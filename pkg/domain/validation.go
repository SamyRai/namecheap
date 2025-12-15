package domain

import (
	"zonekit/pkg/validation"
)

// ValidateDomain validates a domain name format.
func ValidateDomain(domain string) error {
	return validation.ValidateDomain(domain)
}
