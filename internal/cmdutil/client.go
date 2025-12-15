package cmdutil

import (
	"fmt"

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

// DisplayAccountInfo displays information about the account being used.
func DisplayAccountInfo(accountConfig *config.AccountConfig) {
	if accountConfig == nil {
		return
	}

	description := accountConfig.Description
	if description == "" {
		description = "No description"
	}
	fmt.Printf("Using account: %s (%s)\n", accountConfig.Username, description)
	fmt.Println()
}
