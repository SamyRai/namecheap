package model

// Domain represents a domain with its details
type Domain struct {
	Name       string
	User       string
	Created    string
	Expires    string
	IsExpired  bool
	IsLocked   bool
	AutoRenew  bool
	WhoisGuard string
	IsPremium  bool
	IsOurDNS   bool
}

// ContactInfo represents contact information for domain registration
type ContactInfo struct {
	FirstName      string `yaml:"first_name"`
	LastName       string `yaml:"last_name"`
	Address        string `yaml:"address"`
	City           string `yaml:"city"`
	StateProvince  string `yaml:"state_province"`
	PostalCode     string `yaml:"postal_code"`
	Country        string `yaml:"country"`
	Phone          string `yaml:"phone"`
	Email          string `yaml:"email"`
}

// RegistrationRequest contains all details needed to register a domain
type RegistrationRequest struct {
	DomainName string
	Years      int
	Registrant ContactInfo
	Tech       ContactInfo
	Admin      ContactInfo
	AuxBilling ContactInfo
}
