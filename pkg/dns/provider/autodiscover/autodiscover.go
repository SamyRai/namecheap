package autodiscover

import (
	"fmt"
	"os"
	"path/filepath"

	"zonekit/pkg/dns/provider/builder"
	"zonekit/pkg/dns/provider/openapi"
	dnsprovider "zonekit/pkg/dns/provider"
)

// DiscoverAndRegister discovers all providers from subdirectories and registers them
// Scans pkg/dns/provider/*/ directories for openapi.yaml files (OpenAPI-only approach)
func DiscoverAndRegister(baseDir string) error {
	if baseDir == "" {
		baseDir = findProviderDirectory()
		if baseDir == "" {
			return fmt.Errorf("provider directory not found")
		}
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("failed to read provider directory: %w", err)
	}

	var errors []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories and special directories
		name := entry.Name()
		if name[0] == '.' || name == "auth" || name == "builder" || name == "config" ||
		   name == "http" || name == "mapper" || name == "rest" || name == "autodiscover" {
			continue
		}

		// Skip namecheap - it's registered separately via namecheap.Register()
		if name == "namecheap" {
			continue
		}

		providerDir := filepath.Join(baseDir, name)

		// OpenAPI-only approach: require openapi.yaml
		specPath, err := openapi.FindSpecFile(providerDir)
		if err != nil {
			// No OpenAPI spec found, skip this provider
			// (OpenAPI-only approach - no fallback to config.yaml)
			continue
		}

		// Load OpenAPI spec and convert to config
		spec, err := openapi.LoadSpec(specPath)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to load OpenAPI spec for %s: %w", name, err))
			continue
		}

		cfg, err := spec.ToProviderConfig(name)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to convert OpenAPI spec for %s: %w", name, err))
			continue
		}

		provider, err := builder.BuildProvider(cfg)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to build %s provider: %w", name, err))
			continue
		}

		if err := dnsprovider.Register(provider); err != nil {
			// Provider might already be registered, that's okay
			continue
		}
	}

	// Return first error if any, but don't fail completely
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// findProviderDirectory finds the provider directory
func findProviderDirectory() string {
	// Try multiple locations
	locations := []string{
		"pkg/dns/provider",                    // From project root
		"./pkg/dns/provider",                  // Relative to current dir
		filepath.Join("..", "pkg", "dns", "provider"), // From internal dirs
	}

	// Also try to find it relative to the executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		locations = append(locations,
			filepath.Join(execDir, "pkg", "dns", "provider"),
			filepath.Join(filepath.Dir(execDir), "pkg", "dns", "provider"),
		)
	}

	for _, loc := range locations {
		if info, err := os.Stat(loc); err == nil && info.IsDir() {
			return loc
		}
	}

	return ""
}

