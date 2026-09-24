package cmdutil

import (
	"fmt"
	"os"

	"zonekit/pkg/client"
	"zonekit/pkg/config"
)

// CreateClient creates a client from an account configuration.
func CreateClient(accountConfig *config.AccountConfig) (*client.Client, error) {
	if accountConfig == nil {
		return nil, fmt.Errorf("account configuration is nil")
	}

	ncClient, err := client.NewClient(accountConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return ncClient, nil
}

// DisplayAccountInfo prints which account is being used. This is a
// diagnostic, not a result (O2): it always goes to stderr so stdout stays
// clean for scripted/--output json|yaml consumers.
func DisplayAccountInfo(accountConfig *config.AccountConfig) {
	if accountConfig == nil {
		return
	}

	description := accountConfig.Description
	if description == "" {
		description = "No description"
	}
	fmt.Fprintf(os.Stderr, "Using account: %s (%s)\n", accountConfig.Username, description)
	fmt.Fprintln(os.Stderr)
}
