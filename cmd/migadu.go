package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"zonekit/internal/cmdutil"
	"zonekit/pkg/dns"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/mail/migadu"
	"zonekit/pkg/plugin/service"
)

// migaduCmd groups provider-backed Migadu operations. These differ from
// `service setup migadu` in that the record values come from Migadu's own API
// rather than the static template, which removes the manual step of copying a
// per-domain verification token out of the admin console.
var migaduCmd = &cobra.Command{
	Use:   "migadu",
	Short: "Provider-backed Migadu email setup",
	Long: `Configure Migadu email hosting using Migadu's own API as the source of truth.

Unlike the static service template, these commands read each domain's DNS
requirements -- including its unique verification token, which is not derivable
and appears nowhere else in the API -- directly from Migadu, publish them to the
DNS provider, and then run Migadu's diagnostics and activation.

Credentials come from MIGADU_ACCOUNT (the account email, not a mailbox) and
MIGADU_API_KEY (My Account -> API Keys).`,
}

// migaduClient builds a client from the environment, failing with guidance
// rather than a bare auth error later.
func migaduClient() (*migadu.Client, error) {
	account := strings.TrimSpace(os.Getenv("MIGADU_ACCOUNT"))
	key := strings.TrimSpace(os.Getenv("MIGADU_API_KEY"))
	if account == "" || key == "" {
		return nil, fmt.Errorf(
			"MIGADU_ACCOUNT and MIGADU_API_KEY must be set\n" +
				"  MIGADU_ACCOUNT is the Migadu account email (not a mailbox address)\n" +
				"  MIGADU_API_KEY is generated under My Account -> API Keys")
	}
	return migadu.New(account, key), nil
}

var migaduStatusCmd = &cobra.Command{
	Use:   "status [domain]",
	Short: "Show Migadu state and DNS diagnostics for one or all domains",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := migaduClient()
		if err != nil {
			return err
		}
		ctx := context.Background()

		var domains []migadu.Domain
		if len(args) == 1 {
			d, err := client.GetDomain(ctx, args[0])
			if err != nil {
				return err
			}
			domains = []migadu.Domain{*d}
		} else {
			domains, err = client.ListDomains(ctx)
			if err != nil {
				return err
			}
		}

		fmt.Printf("%-26s %-10s %-9s %s\n", "DOMAIN", "STATE", "CAN-SEND", "DIAGNOSTICS")
		for _, d := range domains {
			detail := "-"
			if diag, err := client.GetDiagnostics(ctx, d.Name); err == nil {
				if diag.OK() {
					detail = "ok"
				} else {
					detail = strings.Join(diag.Failed(), " ")
				}
			}
			fmt.Printf("%-26s %-10s %-9v %s\n", d.Name, d.State, d.CanSend, detail)
		}
		return nil
	},
}

var migaduOnboardCmd = &cobra.Command{
	Use:   "onboard <domain>",
	Short: "Publish Migadu DNS from the provider API, then verify and activate",
	Long: `Take a domain from registered to sending mail in one command.

Reads the domain's required records from Migadu, reconciles them against the
live zone (preserving every unrelated record and extending rather than
replacing existing SPF/DMARC policies), publishes, runs Migadu's diagnostics,
and activates the domain once they pass.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		replace, _ := cmd.Flags().GetBool("replace")
		activate, _ := cmd.Flags().GetBool("activate")

		client, err := migaduClient()
		if err != nil {
			return err
		}
		ctx := context.Background()

		bundle, err := client.GetDNSBundle(ctx, domain)
		if err != nil {
			return fmt.Errorf("read Migadu DNS requirements for %s: %w", domain, err)
		}
		desired := bundle.ToRecords()

		accountConfig, err := GetCurrentAccount()
		if err != nil {
			return fmt.Errorf("failed to get account configuration: %w", err)
		}
		dnsClient, err := cmdutil.CreateClient(accountConfig)
		if err != nil {
			return err
		}
		cmdutil.DisplayAccountInfo(accountConfig)
		dnsService := dns.NewService(dnsClient)

		existing, err := dnsService.GetRecords(domain)
		if err != nil {
			return fmt.Errorf("read existing zone for %s: %w", domain, err)
		}

		plan := service.PlanZone(existing, desired, service.PlanOptions{
			Replace:     replace,
			ForcePolicy: false,
		})

		fmt.Printf("\nMigadu records for %s (from provider API)\n", domain)
		fmt.Println("=====================================")
		for _, r := range plan.Desired {
			pref := ""
			if r.RecordType == dnsrecord.RecordTypeMX && r.MXPref > 0 {
				pref = fmt.Sprintf(" (priority: %d)", r.MXPref)
			}
			fmt.Printf("  %s %s → %s%s\n", r.HostName, r.RecordType, r.Address, pref)
		}

		if len(plan.Superseded) > 0 {
			if !replace {
				fmt.Println("\nConflicting records found:")
				for _, c := range plan.Conflicts() {
					fmt.Printf("   - %s\n", c)
				}
				fmt.Println("\nUse --replace to supersede them.")
				return nil
			}
			fmt.Println("\nRecords to be replaced:")
			for _, r := range plan.Superseded {
				fmt.Printf("  %s %s → %s\n", r.HostName, r.RecordType, r.Address)
			}
		}
		fmt.Printf("\nPreserving %d unrelated record(s) in the zone.\n", len(plan.Preserved))

		if dryRun {
			fmt.Println("\nDry run completed. Use without --dry-run to apply changes.")
			return nil
		}

		if err := dnsService.SetRecords(domain, plan.Records(replace)); err != nil {
			return fmt.Errorf("publish DNS for %s: %w", domain, err)
		}
		fmt.Printf("\nPublished Migadu DNS records for %s\n", domain)

		diag, err := client.GetDiagnostics(ctx, domain)
		if err != nil {
			return fmt.Errorf("run Migadu diagnostics: %w", err)
		}
		if !diag.OK() {
			// DNS propagation is the usual cause; say what failed rather than
			// leaving the operator to guess.
			fmt.Printf("\nMigadu diagnostics not passing yet: %s\n", strings.Join(diag.Failed(), " "))
			fmt.Println("This is usually DNS propagation. Re-run with --activate once it clears.")
			return nil
		}
		fmt.Println("Migadu diagnostics: ok")

		if !activate {
			fmt.Printf("Domain is ready to activate. Re-run with --activate, or: zonekit migadu activate %s\n", domain)
			return nil
		}
		d, err := client.Activate(ctx, domain)
		if err != nil {
			return fmt.Errorf("activate %s: %w", domain, err)
		}
		fmt.Printf("Activated %s: state=%s can_send=%v can_receive=%v\n",
			d.Name, d.State, d.CanSend, d.CanReceive)
		return nil
	},
}

var migaduActivateCmd = &cobra.Command{
	Use:   "activate <domain>",
	Short: "Run Migadu diagnostics and activate a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := migaduClient()
		if err != nil {
			return err
		}
		ctx := context.Background()

		diag, err := client.GetDiagnostics(ctx, args[0])
		if err != nil {
			return err
		}
		if !diag.OK() {
			return fmt.Errorf("Migadu DNS checks not passing: %s", strings.Join(diag.Failed(), " "))
		}
		d, err := client.Activate(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Activated %s: state=%s can_send=%v can_receive=%v\n",
			d.Name, d.State, d.CanSend, d.CanReceive)
		return nil
	},
}

var migaduMailboxCmd = &cobra.Command{
	Use:   "mailbox <domain>",
	Short: "List mailboxes on a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := migaduClient()
		if err != nil {
			return err
		}
		boxes, err := client.ListMailboxes(context.Background(), args[0])
		if err != nil {
			return err
		}
		if len(boxes) == 0 {
			fmt.Println("(no mailboxes)")
			return nil
		}
		for _, m := range boxes {
			fmt.Printf("  %-34s active=%-5v send=%-5v receive=%v\n",
				m.Address, m.IsActive, m.MaySend, m.MayReceive)
		}
		return nil
	},
}

var migaduCreateMailboxCmd = &cobra.Command{
	Use:   "create-mailbox <domain> <local-part>",
	Short: "Create a mailbox using Migadu's invitation flow",
	Long: `Create a mailbox without handling a password.

The mailbox is created with Migadu's invitation flow: the recipient sets the
password themselves, so no credential is generated, transported or logged by
this tool. Use --send-only for service accounts such as noreply@.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, localPart := args[0], args[1]
		recovery, _ := cmd.Flags().GetString("recovery-email")
		name, _ := cmd.Flags().GetString("name")
		sendOnly, _ := cmd.Flags().GetBool("send-only")

		if recovery == "" {
			return fmt.Errorf("--recovery-email is required: it receives the invitation used to set the password")
		}

		client, err := migaduClient()
		if err != nil {
			return err
		}
		spec := migadu.MailboxSpec{
			LocalPart:     localPart,
			Name:          name,
			RecoveryEmail: recovery,
			MaySend:       true,
			MayReceive:    !sendOnly,
			MayAccessIMAP: !sendOnly,
			MayAccessPOP3: !sendOnly,
		}
		m, err := client.CreateMailbox(context.Background(), domain, spec)
		if err != nil {
			return err
		}
		fmt.Printf("Created %s (active=%v). An invitation was sent to %s to set the password.\n",
			m.Address, m.IsActive, recovery)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migaduCmd)
	migaduCmd.AddCommand(migaduStatusCmd)
	migaduCmd.AddCommand(migaduOnboardCmd)
	migaduCmd.AddCommand(migaduActivateCmd)
	migaduCmd.AddCommand(migaduMailboxCmd)
	migaduCmd.AddCommand(migaduCreateMailboxCmd)

	migaduOnboardCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	migaduOnboardCmd.Flags().Bool("replace", false, "Supersede conflicting records")
	migaduOnboardCmd.Flags().Bool("activate", false, "Activate the domain once diagnostics pass")

	migaduCreateMailboxCmd.Flags().String("recovery-email", "", "Address that receives the password invitation (required)")
	migaduCreateMailboxCmd.Flags().String("name", "", "Display name for the mailbox")
	migaduCreateMailboxCmd.Flags().Bool("send-only", false, "Service account: sending only, no IMAP/POP3 or receiving")
}
