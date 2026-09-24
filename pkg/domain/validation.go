package domain

import (
	"go.glpx.pro/zonekit/pkg/validation"
)

// ValidateDomain validates a domain name format.
func ValidateDomain(domain string) error {
	return validation.ValidateDomain(domain)
}
