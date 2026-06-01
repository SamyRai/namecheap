package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"zonekit/pkg/config"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Manage authentication for DNS providers, including OAuth flows.`,
}

// authLoginCmd represents the auth login command
var authLoginCmd = &cobra.Command{
	Use:   "login <provider>",
	Short: "Login to a provider using OAuth",
	Long:  `Start an OAuth 2.0 flow to authenticate with a supported provider.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := args[0]
		
		fmt.Printf("Starting OAuth login flow for provider: %s\n", providerName)

		// Placeholder OAuth config
		// In a real implementation, we would look up the Endpoint, ClientID, and ClientSecret for the provider.
		// For example, DigitalOcean, Google Cloud, etc.
		var oauthConfig *oauth2.Config

		if providerName == "google" {
			oauthConfig = &oauth2.Config{
				ClientID:     os.Getenv("ZONEKIT_GOOGLE_CLIENT_ID"),
				ClientSecret: os.Getenv("ZONEKIT_GOOGLE_CLIENT_SECRET"),
				Scopes:       []string{"https://www.googleapis.com/auth/ndev.clouddns.readwrite"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://accounts.google.com/o/oauth2/auth",
					TokenURL: "https://oauth2.googleapis.com/token",
				},
				RedirectURL: "http://localhost:8080/callback",
			}
		} else {
			return fmt.Errorf("OAuth login not yet implemented for provider: %s", providerName)
		}

		if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
			return fmt.Errorf("OAuth Client ID and Secret must be configured in environment variables")
		}

		// Channel to receive the authorization code
		codeChan := make(chan string)
		errChan := make(chan error)

		// Start local server to receive the callback
		server := &http.Server{Addr: ":8080"}
		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				errStr := r.URL.Query().Get("error")
				errChan <- fmt.Errorf("OAuth error: %s", errStr)
				fmt.Fprintf(w, "OAuth error: %s. You can close this window.", errStr)
				return
			}
			
			fmt.Fprintf(w, "Authentication successful! You can close this window and return to the terminal.")
			codeChan <- code
		})

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("failed to start local server: %w", err)
			}
		}()

		// Generate authorization URL
		url := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		fmt.Printf("\nPlease open the following URL in your browser to authenticate:\n\n%s\n\n", url)
		fmt.Println("Waiting for authentication...")

		// Wait for code or error
		var code string
		select {
		case code = <-codeChan:
			fmt.Println("Received authorization code.")
		case err := <-errChan:
			server.Shutdown(context.Background())
			return err
		}

		// Shutdown server
		server.Shutdown(context.Background())

		// Exchange code for token
		token, err := oauthConfig.Exchange(cmd.Context(), code)
		if err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}

		// Save token to config keyring
		configManager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to get config manager: %w", err)
		}

		accountName := configManager.GetCurrentAccountName()
		accountConfig, err := configManager.GetCurrentAccount()
		if err != nil {
			accountConfig = &config.AccountConfig{Provider: providerName}
			configManager.AddAccount(accountName, accountConfig)
		}

		// For OAuth, we store the AccessToken in APIKey (or we can use a new field).
		// Wait, OAuth tokens require refreshing. The go-keyring can store a JSON string with the full token.
		// For now, we'll store the AccessToken in APIKey. In the future, a dedicated Token store could be built.
		accountConfig.APIKey = token.AccessToken
		
		err = configManager.SaveSecretsToKeyring(accountName, accountConfig)
		if err != nil {
			return fmt.Errorf("failed to save token to keyring: %w", err)
		}

		fmt.Println("Successfully authenticated and saved credentials to Keyring.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
}
