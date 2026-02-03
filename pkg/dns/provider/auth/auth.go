package auth

import (
	"fmt"
	"os"
	"strings"
)

// Method represents authentication method
type Method string

const (
	MethodAPIKey Method = "api_key"
	MethodOAuth  Method = "oauth"
	MethodBasic  Method = "basic"
	MethodBearer Method = "bearer"
	MethodCustom Method = "custom"
)

// Credentials holds authentication credentials
type Credentials map[string]interface{}

// Authenticator handles authentication for HTTP requests
type Authenticator interface {
	// GetHeaders returns headers to add to requests
	GetHeaders() map[string]string
	// Validate checks if credentials are valid
	Validate() error
}

// NewAuthenticator creates an authenticator based on method and credentials
func NewAuthenticator(method string, credentials Credentials) (Authenticator, error) {
	switch Method(method) {
	case MethodAPIKey:
		return NewAPIKeyAuthenticator(credentials)
	case MethodBearer:
		return NewBearerAuthenticator(credentials)
	case MethodBasic:
		return NewBasicAuthenticator(credentials)
	case MethodOAuth:
		return NewOAuthAuthenticator(credentials)
	case MethodCustom:
		return NewCustomAuthenticator(credentials)
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", method)
	}
}

// APIKeyAuthenticator handles API key authentication
type APIKeyAuthenticator struct {
	APIKey string
	Email  string // Some providers use email + API key
	Header string // Header name (e.g., "X-API-Key", "Authorization")
}

// NewAPIKeyAuthenticator creates an API key authenticator
func NewAPIKeyAuthenticator(credentials Credentials) (*APIKeyAuthenticator, error) {
	apiKey, ok := credentials["api_key"].(string)
	if !ok {
		apiKey = getEnvOrValue(credentials["api_key"])
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for api_key authentication")
	}

	email := getEnvOrValue(credentials["email"])
	header := getStringValue(credentials["header"], "X-API-Key")

	return &APIKeyAuthenticator{
		APIKey: apiKey,
		Email:  email,
		Header: header,
	}, nil
}

func (a *APIKeyAuthenticator) GetHeaders() map[string]string {
	headers := make(map[string]string)

	if a.Email != "" {
		headers["X-Auth-Email"] = a.Email
	}

	if a.Header == "Authorization" {
		headers["Authorization"] = "Bearer " + a.APIKey
	} else {
		headers[a.Header] = a.APIKey
	}

	return headers
}

func (a *APIKeyAuthenticator) Validate() error {
	if a.APIKey == "" {
		return fmt.Errorf("API key is empty")
	}
	return nil
}

// BearerAuthenticator handles Bearer token authentication
type BearerAuthenticator struct {
	Token string
}

// NewBearerAuthenticator creates a Bearer token authenticator
func NewBearerAuthenticator(credentials Credentials) (*BearerAuthenticator, error) {
	token, ok := credentials["token"].(string)
	if !ok {
		token = getEnvOrValue(credentials["token"])
	}
	if token == "" {
		return nil, fmt.Errorf("token is required for bearer authentication")
	}

	return &BearerAuthenticator{Token: token}, nil
}

func (a *BearerAuthenticator) GetHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + a.Token,
	}
}

func (a *BearerAuthenticator) Validate() error {
	if a.Token == "" {
		return fmt.Errorf("bearer token is empty")
	}
	return nil
}

// BasicAuthenticator handles Basic authentication
type BasicAuthenticator struct {
	Username string
	Password string
}

// NewBasicAuthenticator creates a Basic authenticator
func NewBasicAuthenticator(credentials Credentials) (*BasicAuthenticator, error) {
	username := getEnvOrValue(credentials["username"])
	password := getEnvOrValue(credentials["password"])

	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required for basic authentication")
	}

	return &BasicAuthenticator{
		Username: username,
		Password: password,
	}, nil
}

func (a *BasicAuthenticator) GetHeaders() map[string]string {
	// Basic auth is typically handled by http.Client, but we can add it here if needed
	// For now, return empty - caller should use http.Client's Transport
	return map[string]string{}
}

func (a *BasicAuthenticator) Validate() error {
	if a.Username == "" || a.Password == "" {
		return fmt.Errorf("username or password is empty")
	}
	return nil
}

// OAuthAuthenticator handles OAuth authentication (placeholder for future)
type OAuthAuthenticator struct {
	AccessToken string
}

// NewOAuthAuthenticator creates an OAuth authenticator
func NewOAuthAuthenticator(credentials Credentials) (*OAuthAuthenticator, error) {
	// OAuth implementation would go here
	// For now, treat it like Bearer token
	token := getEnvOrValue(credentials["access_token"])
	if token == "" {
		return nil, fmt.Errorf("access_token is required for oauth authentication")
	}

	return &OAuthAuthenticator{AccessToken: token}, nil
}

func (a *OAuthAuthenticator) GetHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + a.AccessToken,
	}
}

func (a *OAuthAuthenticator) Validate() error {
	if a.AccessToken == "" {
		return fmt.Errorf("OAuth access token is empty")
	}
	return nil
}

// CustomAuthenticator handles custom authentication methods
type CustomAuthenticator struct {
	Headers map[string]string
}

// NewCustomAuthenticator creates a custom authenticator
func NewCustomAuthenticator(credentials Credentials) (*CustomAuthenticator, error) {
	headers := make(map[string]string)

	if headersMap, ok := credentials["headers"].(map[string]interface{}); ok {
		for k, v := range headersMap {
			headers[k] = getEnvOrValue(v)
		}
	}

	return &CustomAuthenticator{Headers: headers}, nil
}

func (a *CustomAuthenticator) GetHeaders() map[string]string {
	return a.Headers
}

func (a *CustomAuthenticator) Validate() error {
	if len(a.Headers) == 0 {
		return fmt.Errorf("custom authenticator requires at least one header")
	}
	return nil
}

// Helper functions

func getEnvOrValue(value interface{}) string {
	if str, ok := value.(string); ok {
		// Check if it's an environment variable reference
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") {
			envVar := strings.TrimPrefix(strings.TrimSuffix(str, "}"), "${")
			return os.Getenv(envVar)
		}
		return str
	}
	return ""
}

func getStringValue(value interface{}, defaultValue string) string {
	if str := getEnvOrValue(value); str != "" {
		return str
	}
	return defaultValue
}
