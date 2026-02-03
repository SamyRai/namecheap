package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dnsprovider "zonekit/pkg/dns/provider"

	"gopkg.in/yaml.v3"
)

// Spec represents a parsed OpenAPI specification
type Spec struct {
	OpenAPI    string                 `yaml:"openapi" json:"openapi"`
	Info       Info                   `yaml:"info" json:"info"`
	Servers    []Server               `yaml:"servers" json:"servers"`
	Paths      map[string]interface{} `yaml:"paths" json:"paths"`
	Components *Components            `yaml:"components" json:"components"`
}

// Info contains API metadata
type Info struct {
	Title   string `yaml:"title" json:"title"`
	Version string `yaml:"version" json:"version"`
}

// Server represents an API server
type Server struct {
	URL string `yaml:"url" json:"url"`
}

// Components contains reusable OpenAPI components
type Components struct {
	Schemas         map[string]interface{} `yaml:"schemas" json:"schemas"`
	SecuritySchemes map[string]interface{} `yaml:"securitySchemes" json:"securitySchemes"`
}

// LoadSpec loads an OpenAPI specification from a file
func LoadSpec(filePath string) (*Spec, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec: %w", err)
	}

	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		// Try JSON if YAML fails
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
		}
	}

	return &spec, nil
}

// FindSpecFile looks for OpenAPI spec files in a directory
func FindSpecFile(dirPath string) (string, error) {
	possibleNames := []string{
		"openapi.yaml",
		"openapi.yml",
		"openapi.json",
		"swagger.yaml",
		"swagger.yml",
		"swagger.json",
	}

	for _, name := range possibleNames {
		path := filepath.Join(dirPath, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no OpenAPI spec file found in %s", dirPath)
}

// ToProviderConfig converts an OpenAPI spec to a provider config
func (s *Spec) ToProviderConfig(providerName string) (*dnsprovider.Config, error) {
	cfg := &dnsprovider.Config{
		Name:        providerName,
		DisplayName: s.Info.Title,
		Type:        "rest",
	}

	// Extract base URL from servers
	if len(s.Servers) > 0 {
		cfg.API.BaseURL = s.Servers[0].URL
	}

	// Extract endpoints from paths
	cfg.API.Endpoints = s.extractEndpoints()

	// Extract authentication from security schemes
	if s.Components != nil && s.Components.SecuritySchemes != nil {
		authMethod, credentials := s.extractAuthentication()
		cfg.Auth.Method = authMethod
		cfg.Auth.Credentials = credentials
	}

	// Extract field mappings from schemas
	if s.Components != nil && s.Components.Schemas != nil {
		cfg.Mappings = s.extractMappings()
	}

	// Set defaults
	if cfg.API.Timeout == 0 {
		cfg.API.Timeout = 30
	}
	if cfg.API.Retries == 0 {
		cfg.API.Retries = 3
	}

	return cfg, nil
}

// extractEndpoints extracts DNS operation endpoints from OpenAPI paths
func (s *Spec) extractEndpoints() map[string]string {
	endpoints := make(map[string]string)

	for path, pathItem := range s.Paths {
		pathMap, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}

		// Map HTTP methods to DNS operations
		for method, operation := range pathMap {
			opMap, ok := operation.(map[string]interface{})
			if !ok {
				continue
			}

			operationID, _ := opMap["operationId"].(string)
			endpointKey := s.mapOperationToEndpoint(method, operationID, path)
			if endpointKey != "" {
				// Avoid overwriting existing endpoints with single-item paths (prefer list endpoints)
				if existing, ok := endpoints[endpointKey]; ok && existing != "" {
					// Prefer the endpoint without path parameters
					if strings.Contains(existing, "{") && !strings.Contains(path, "{") {
						endpoints[endpointKey] = path
					}
					// otherwise keep existing
				} else {
					endpoints[endpointKey] = path
				}
			}
		}
	}

	return endpoints
}

// mapOperationToEndpoint maps OpenAPI operations to our endpoint keys
func (s *Spec) mapOperationToEndpoint(method, operationID, path string) string {
	method = strings.ToLower(method)
	operationID = strings.ToLower(operationID)
	path = strings.ToLower(path)

	// Try to infer from operation ID
	if strings.Contains(operationID, "list") || strings.Contains(operationID, "get") {
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "get_records"
		}
	}
	if strings.Contains(operationID, "create") || strings.Contains(operationID, "add") {
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "create_record"
		}
	}
	if strings.Contains(operationID, "update") || strings.Contains(operationID, "modify") {
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "update_record"
		}
	}
	if strings.Contains(operationID, "delete") || strings.Contains(operationID, "remove") {
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "delete_record"
		}
	}

	// Fallback to HTTP method
	switch method {
	case "get":
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "get_records"
		}
	case "post":
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "create_record"
		}
	case "put", "patch":
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "update_record"
		}
	case "delete":
		if strings.Contains(path, "record") || strings.Contains(path, "dns") {
			return "delete_record"
		}
	}

	return ""
}

// extractAuthentication extracts authentication method from OpenAPI security schemes
func (s *Spec) extractAuthentication() (string, map[string]interface{}) {
	if s.Components == nil || s.Components.SecuritySchemes == nil {
		return "", nil
	}

	credentials := make(map[string]interface{})

	for name, scheme := range s.Components.SecuritySchemes {
		schemeMap, ok := scheme.(map[string]interface{})
		if !ok {
			continue
		}

		schemeType, _ := schemeMap["type"].(string)
		schemeType = strings.ToLower(schemeType)

		switch schemeType {
		case "apikey":
			// API Key authentication
			in, _ := schemeMap["in"].(string)
			keyName, _ := schemeMap["name"].(string)

			if in == "header" {
				credentials["api_key"] = fmt.Sprintf("${%s_API_KEY}", strings.ToUpper(name))
				if keyName != "" {
					credentials["header_name"] = keyName
				}
				return "api_key", credentials
			}

		case "http":
			// HTTP authentication (Bearer, Basic)
			scheme, _ := schemeMap["scheme"].(string)
			scheme = strings.ToLower(scheme)

			if scheme == "bearer" {
				credentials["token"] = fmt.Sprintf("${%s_API_TOKEN}", strings.ToUpper(name))
				return "bearer", credentials
			}
			if scheme == "basic" {
				credentials["username"] = fmt.Sprintf("${%s_USERNAME}", strings.ToUpper(name))
				credentials["password"] = fmt.Sprintf("${%s_PASSWORD}", strings.ToUpper(name))
				return "basic", credentials
			}

		case "oauth2":
			// OAuth2 authentication
			credentials["token"] = fmt.Sprintf("${%s_OAUTH_TOKEN}", strings.ToUpper(name))
			return "oauth", credentials
		}
	}

	return "", nil
}

// extractMappings extracts field mappings from OpenAPI schemas
func (s *Spec) extractMappings() *dnsprovider.FieldMappings {
	if s.Components == nil || s.Components.Schemas == nil {
		return nil
	}

	mappings := &dnsprovider.FieldMappings{}

	// Look for DNS record schema
	for schemaName, schema := range s.Components.Schemas {
		origName := schemaName
		schemaName = strings.ToLower(schemaName)
		if !strings.Contains(schemaName, "record") && !strings.Contains(schemaName, "dns") {
			continue
		}

		schemaMap, ok := schema.(map[string]interface{})
		if !ok {
			continue
		}

		properties, ok := schemaMap["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		// Map common DNS record fields
		for propName := range properties {
			propLower := strings.ToLower(propName)
			switch propLower {
			case "name", "hostname", "host":
				mappings.Request.HostName = propName
				mappings.Response.HostName = propName
			case "type", "recordtype", "record_type":
				mappings.Request.RecordType = propName
				mappings.Response.RecordType = propName
			case "content", "data", "value", "address":
				mappings.Request.Address = propName
				mappings.Response.Address = propName
			case "ttl":
				mappings.Request.TTL = propName
				mappings.Response.TTL = propName
			case "priority", "preference", "mxpref", "mx_pref":
				mappings.Request.MXPref = propName
				mappings.Response.MXPref = propName
			case "id", "recordid", "record_id", "_id":
				mappings.Request.ID = propName
				mappings.Response.ID = propName
			}
		}

		// Try to find list path by inspecting other schemas for arrays of this schema
		for _, otherSchema := range s.Components.Schemas {
			otherSchemaMap, ok := otherSchema.(map[string]interface{})
			if !ok {
				continue
			}

			// Look for properties that are arrays with items referencing this schema
			if props, ok := otherSchemaMap["properties"].(map[string]interface{}); ok {
				for propName, prop := range props {
					propMap, ok := prop.(map[string]interface{})
					if !ok {
						continue
					}

					if propMap["type"] == "array" {
						if items, ok := propMap["items"].(map[string]interface{}); ok {
							if ref, ok := items["$ref"].(string); ok {
								refLower := strings.ToLower(ref)
								if strings.Contains(refLower, strings.ToLower(origName)) || strings.HasSuffix(refLower, "/"+strings.ToLower(origName)) {
									mappings.ListPath = propName
									break
								}
							}
						}
					}
				}
			}
			if mappings.ListPath != "" {
				break
			}
		}
	}

	return mappings
}
