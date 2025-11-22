package dns

// Bulk operation actions
const (
	BulkActionAdd    = "add"
	BulkActionUpdate = "update"
	BulkActionDelete = "delete"
)

// Default TTL values
const (
	DefaultTTL    = 1800  // 30 minutes
	MinTTL        = 60    // 1 minute
	MaxTTL        = 86400 // 24 hours
	DefaultMXPref = 10
	MinMXPref     = 0
	MaxMXPref     = 65535
)

// Email type for DNS records
const (
	EmailTypeMX = "MX"
)
