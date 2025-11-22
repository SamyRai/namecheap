package testutil

// TestAccountConfig represents account configuration for testing
type TestAccountConfig struct {
	Username    string
	APIUser     string
	APIKey      string
	ClientIP    string
	UseSandbox  bool
	Description string
}

// AccountConfigFixture returns a valid test account configuration data
func AccountConfigFixture() TestAccountConfig {
	return TestAccountConfig{
		Username:    "testuser",
		APIUser:     "testapiuser",
		APIKey:      "testapikey",
		ClientIP:    "127.0.0.1",
		UseSandbox:  true,
		Description: "Test account",
	}
}

// AccountConfigFixtureWithValues returns test account configuration data with custom values
func AccountConfigFixtureWithValues(username, apiUser, apiKey, clientIP string, useSandbox bool) TestAccountConfig {
	return TestAccountConfig{
		Username:    username,
		APIUser:     apiUser,
		APIKey:      apiKey,
		ClientIP:    clientIP,
		UseSandbox:  useSandbox,
		Description: "Test account",
	}
}

// TestDNSRecord represents DNS record for testing
type TestDNSRecord struct {
	HostName   string
	RecordType string
	Address    string
	TTL        int
	MXPref     int
}

// DNSRecordFixture returns a valid test DNS record data
func DNSRecordFixture() TestDNSRecord {
	return TestDNSRecord{
		HostName:   "@",
		RecordType: "A",
		Address:    "192.168.1.1",
		TTL:        1800,
		MXPref:     0,
	}
}

// DNSRecordFixtureWithValues returns test DNS record data with custom values
func DNSRecordFixtureWithValues(hostname, recordType, address string, ttl, mxPref int) TestDNSRecord {
	return TestDNSRecord{
		HostName:   hostname,
		RecordType: recordType,
		Address:    address,
		TTL:        ttl,
		MXPref:     mxPref,
	}
}

// ValidDomainFixture returns a valid test domain name
func ValidDomainFixture() string {
	return "example.com"
}

// ValidSubdomainFixture returns a valid test subdomain
func ValidSubdomainFixture() string {
	return "www.example.com"
}

