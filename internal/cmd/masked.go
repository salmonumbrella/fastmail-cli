package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newMaskedCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "masked",
		Aliases: []string{"mask", "alias"},
		Short:   "Masked email (alias) operations",
		Long: `Manage Fastmail masked email addresses.

Masked emails are disposable aliases that forward to your inbox.
Use them for signups to protect your real email address.`,
	}

	cmd.AddCommand(newMaskedListCmd(flags))
	cmd.AddCommand(newMaskedCreateCmd(flags))
	cmd.AddCommand(newMaskedGetCmd(flags))
	cmd.AddCommand(newMaskedEnableCmd(flags))
	cmd.AddCommand(newMaskedDisableCmd(flags))
	cmd.AddCommand(newMaskedDeleteCmd(flags))
	cmd.AddCommand(newMaskedDescriptionCmd(flags))

	return cmd
}

func newMaskedListCmd(flags *rootFlags) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list [domain]",
		Short: "List masked emails",
		Long: `List masked emails, optionally filtered by domain.

Without a domain argument, lists all masked emails.
With a domain, lists only aliases for that domain.`,
		Example: `  fastmail masked list
  fastmail masked list example.com`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			var aliases []jmap.MaskedEmail
			var domain string

			if len(args) > 0 {
				domain = args[0]
				aliases, err = client.GetMaskedEmailsForDomain(cmd.Context(), domain)
			} else {
				aliases, err = client.GetMaskedEmails(cmd.Context())
				// Filter out deleted unless --all
				if !all {
					var filtered []jmap.MaskedEmail
					for _, a := range aliases {
						if a.State != jmap.MaskedEmailDeleted {
							filtered = append(filtered, a)
						}
					}
					aliases = filtered
				}
			}

			if err != nil {
				return fmt.Errorf("failed to list masked emails: %w", err)
			}

			// Sort by domain, then email
			sort.Slice(aliases, func(i, j int) bool {
				if aliases[i].ForDomain != aliases[j].ForDomain {
					return aliases[i].ForDomain < aliases[j].ForDomain
				}
				return aliases[i].Email < aliases[j].Email
			})

			if isJSON(cmd.Context()) {
				return printJSON(cmd, aliases)
			}

			if len(aliases) == 0 {
				if domain != "" {
					outfmt.Errorf("No masked emails found for %s", domain)
				} else {
					outfmt.Errorf("No masked emails found")
				}
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "EMAIL\tDOMAIN\tSTATE\tDESCRIPTION")
			for _, alias := range aliases {
				desc := alias.Description
				if desc == "" {
					desc = "-"
				}
				// Truncate description for display
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				domain := alias.ForDomain
				if domain == "" {
					domain = "-"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					alias.Email,
					sanitizeTab(domain),
					alias.State,
					sanitizeTab(desc),
				)
			}
			tw.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Include deleted aliases")

	return cmd
}

func newMaskedCreateCmd(flags *rootFlags) *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "create <domain> [description]",
		Short: "Create or get a masked email for a domain",
		Long: `Create a new masked email alias for a domain.

If an alias already exists for the domain, it returns the existing one.
The domain is normalized (paths and ports are stripped).`,
		Example: `  fastmail masked create example.com
  fastmail masked create example.com "Shopping account"
  fastmail masked create https://shop.example.com/signup`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			domain := args[0]

			// Check if positional description provided
			if len(args) > 1 {
				description = args[1]
			}

			// Validate input is a domain, not an email
			if jmap.LooksLikeEmail(domain) {
				return fmt.Errorf("expected a domain, got an email address: %s\nUse 'masked get' to lookup an existing alias", domain)
			}

			// Check for existing aliases first
			existing, err := client.GetMaskedEmailsForDomain(cmd.Context(), domain)
			if err != nil {
				return fmt.Errorf("failed to check existing aliases: %w", err)
			}

			// If alias exists, return it (select best one by state)
			if len(existing) > 0 {
				best := selectBestAlias(existing)

				if isJSON(cmd.Context()) {
					return printJSON(cmd, map[string]any{
						"alias":   best,
						"created": false,
					})
				}

				if len(existing) > 1 {
					fmt.Printf("Found %d aliases for %s, selected best:\n", len(existing), domain)
				}
				fmt.Printf("%s (state: %s)\n", best.Email, best.State)
				if description != "" {
					fmt.Fprintf(os.Stderr, "Note: description not applied to existing alias. Use 'masked description' to update.\n")
				}
				return nil
			}

			// Create new alias
			alias, err := client.CreateMaskedEmail(cmd.Context(), domain, description)
			if err != nil {
				return fmt.Errorf("failed to create masked email: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"alias":   alias,
					"created": true,
				})
			}

			fmt.Printf("Created: %s (state: %s)\n", alias.Email, alias.State)
			return nil
		},
	}

	return cmd
}

func newMaskedGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <email>",
		Short: "Get details of a masked email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			email := args[0]
			if !jmap.LooksLikeEmail(email) {
				return fmt.Errorf("expected an email address, got: %s\nUse 'masked list' to search by domain", email)
			}

			alias, err := client.GetMaskedEmailByEmail(cmd.Context(), email)
			if err != nil {
				return fmt.Errorf("failed to get masked email: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, alias)
			}

			fmt.Printf("Email:       %s\n", alias.Email)
			fmt.Printf("State:       %s\n", alias.State)
			fmt.Printf("Domain:      %s\n", alias.ForDomain)
			fmt.Printf("Description: %s\n", alias.Description)
			if !alias.CreatedAt.IsZero() {
				fmt.Printf("Created:     %s\n", alias.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			if alias.LastMessageAt != nil {
				fmt.Printf("Last Email:  %s\n", alias.LastMessageAt.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	return cmd
}

func newMaskedEnableCmd(flags *rootFlags) *cobra.Command {
	var domain string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "enable [email]",
		Short: "Enable masked email(s)",
		Long: `Enable a masked email alias or all aliases for a domain.

New aliases start in 'pending' state and are auto-enabled on first email.
If no email is received within 24 hours, pending aliases are deleted.
Use this command to manually enable an alias.`,
		Example: `  fastmail masked enable user.1234@fastmail.com
  fastmail masked enable --domain example.com
  fastmail masked enable --domain example.com --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && domain == "" {
				return fmt.Errorf("either provide an email address or use --domain flag")
			}
			if len(args) > 0 && domain != "" {
				return fmt.Errorf("cannot use both email argument and --domain flag")
			}

			if domain != "" {
				return bulkUpdateMaskedEmailState(cmd, flags, domain, jmap.MaskedEmailEnabled, dryRun)
			}
			return updateMaskedEmailState(cmd, flags, args[0], jmap.MaskedEmailEnabled, dryRun)
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Enable all aliases for this domain")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without making changes")

	return cmd
}

func newMaskedDisableCmd(flags *rootFlags) *cobra.Command {
	var domain string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "disable [email]",
		Short: "Disable masked email(s) (emails go to trash)",
		Long: `Disable a masked email alias or all aliases for a domain.

When disabled, emails sent to the alias are moved to trash.`,
		Example: `  fastmail masked disable user.1234@fastmail.com
  fastmail masked disable --domain example.com
  fastmail masked disable --domain example.com --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && domain == "" {
				return fmt.Errorf("either provide an email address or use --domain flag")
			}
			if len(args) > 0 && domain != "" {
				return fmt.Errorf("cannot use both email argument and --domain flag")
			}

			if domain != "" {
				return bulkUpdateMaskedEmailState(cmd, flags, domain, jmap.MaskedEmailDisabled, dryRun)
			}
			return updateMaskedEmailState(cmd, flags, args[0], jmap.MaskedEmailDisabled, dryRun)
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Disable all aliases for this domain")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without making changes")

	return cmd
}

func newMaskedDeleteCmd(flags *rootFlags) *cobra.Command {
	var domain string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete [email]",
		Short: "Delete masked email(s) (emails will bounce)",
		Long: `Delete a masked email alias or all aliases for a domain.

When deleted, emails sent to the alias will bounce.`,
		Example: `  fastmail masked delete user.1234@fastmail.com
  fastmail masked delete --domain example.com
  fastmail masked delete --domain example.com --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && domain == "" {
				return fmt.Errorf("either provide an email address or use --domain flag")
			}
			if len(args) > 0 && domain != "" {
				return fmt.Errorf("cannot use both email argument and --domain flag")
			}

			if domain != "" {
				return bulkUpdateMaskedEmailState(cmd, flags, domain, jmap.MaskedEmailDeleted, dryRun)
			}
			return updateMaskedEmailState(cmd, flags, args[0], jmap.MaskedEmailDeleted, dryRun)
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Delete all aliases for this domain")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without making changes")

	return cmd
}

func newMaskedDescriptionCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "description <email> <new-description>",
		Short: "Update the description of a masked email",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			email := args[0]
			description := args[1]

			if !jmap.LooksLikeEmail(email) {
				return fmt.Errorf("expected an email address, got: %s", email)
			}

			// Get the alias to find its ID
			alias, err := client.GetMaskedEmailByEmail(cmd.Context(), email)
			if err != nil {
				return fmt.Errorf("failed to get masked email: %w", err)
			}

			if alias.Description == description {
				fmt.Println("Description already set to the requested value")
				return nil
			}

			err = client.UpdateMaskedEmailDescription(cmd.Context(), alias.ID, description)
			if err != nil {
				return fmt.Errorf("failed to update description: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, map[string]any{
					"email":       email,
					"description": description,
					"updated":     true,
				})
			}

			fmt.Println("Description updated")
			return nil
		},
	}

	return cmd
}

func updateMaskedEmailState(cmd *cobra.Command, flags *rootFlags, email string, state jmap.MaskedEmailState, dryRun bool) error {
	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if !jmap.LooksLikeEmail(email) {
		return fmt.Errorf("expected an email address, got: %s", email)
	}

	// Get the alias to find its ID and check current state
	alias, err := client.GetMaskedEmailByEmail(cmd.Context(), email)
	if err != nil {
		return fmt.Errorf("failed to get masked email: %w", err)
	}

	if alias.State == state {
		return fmt.Errorf("alias %s is already %s", email, state)
	}

	// Action verb based on state
	action := stateAction(state)

	if dryRun {
		if isJSON(cmd.Context()) {
			return printJSON(cmd, map[string]any{
				"dry_run":       true,
				"email":         email,
				"current_state": alias.State,
				"new_state":     state,
			})
		}
		fmt.Printf("[dry-run] Would %s: %s (currently %s)\n", action, email, alias.State)
		return nil
	}

	err = client.UpdateMaskedEmailState(cmd.Context(), alias.ID, state)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	if isJSON(cmd.Context()) {
		return printJSON(cmd, map[string]any{
			"email": email,
			"state": state,
		})
	}

	fmt.Printf("Masked email %s %s\n", email, action)
	return nil
}

func bulkUpdateMaskedEmailState(cmd *cobra.Command, flags *rootFlags, domain string, state jmap.MaskedEmailState, dryRun bool) error {
	client, err := getClient(flags)
	if err != nil {
		return err
	}

	// Get all aliases for the domain
	aliases, err := client.GetMaskedEmailsForDomain(cmd.Context(), domain)
	if err != nil {
		return fmt.Errorf("failed to get aliases for domain: %w", err)
	}

	if len(aliases) == 0 {
		return fmt.Errorf("no aliases found for domain: %s", domain)
	}

	// Filter to only aliases that need updating
	var toUpdate []jmap.MaskedEmail
	for _, alias := range aliases {
		if alias.State != state {
			toUpdate = append(toUpdate, alias)
		}
	}

	if len(toUpdate) == 0 {
		fmt.Printf("All %d aliases for %s are already %s\n", len(aliases), domain, state)
		return nil
	}

	action := stateAction(state)

	if dryRun {
		if isJSON(cmd.Context()) {
			results := make([]map[string]any, len(toUpdate))
			for i, alias := range toUpdate {
				results[i] = map[string]any{
					"email":         alias.Email,
					"current_state": alias.State,
					"new_state":     state,
				}
			}
			return printJSON(cmd, map[string]any{
				"dry_run": true,
				"domain":  domain,
				"count":   len(toUpdate),
				"aliases": results,
			})
		}

		fmt.Printf("[dry-run] Would %s %d aliases for %s:\n", action, len(toUpdate), domain)
		for _, alias := range toUpdate {
			fmt.Printf("  %s (currently %s)\n", alias.Email, alias.State)
		}
		return nil
	}

	// Perform the updates
	var succeeded, failed int
	var errors []string

	for _, alias := range toUpdate {
		err := client.UpdateMaskedEmailState(cmd.Context(), alias.ID, state)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("%s: %v", alias.Email, err))
		} else {
			succeeded++
		}
	}

	if isJSON(cmd.Context()) {
		return printJSON(cmd, map[string]any{
			"domain":    domain,
			"state":     state,
			"succeeded": succeeded,
			"failed":    failed,
			"errors":    errors,
		})
	}

	if failed == 0 {
		fmt.Printf("Successfully %s %d aliases for %s\n", action, succeeded, domain)
	} else {
		fmt.Printf("%s %d aliases, %d failed:\n", action, succeeded, failed)
		for _, e := range errors {
			fmt.Printf("  %s\n", e)
		}
	}

	return nil
}

func stateAction(state jmap.MaskedEmailState) string {
	switch state {
	case jmap.MaskedEmailEnabled:
		return "enable"
	case jmap.MaskedEmailDisabled:
		return "disable"
	case jmap.MaskedEmailDeleted:
		return "delete"
	default:
		return "update"
	}
}

// selectBestAlias selects the best alias based on state priority
// Priority: enabled > pending > disabled > deleted
func selectBestAlias(aliases []jmap.MaskedEmail) *jmap.MaskedEmail {
	if len(aliases) == 0 {
		return nil
	}

	statePriority := map[jmap.MaskedEmailState]int{
		jmap.MaskedEmailEnabled:  0,
		jmap.MaskedEmailPending:  1,
		jmap.MaskedEmailDisabled: 2,
		jmap.MaskedEmailDeleted:  3,
	}

	best := &aliases[0]
	bestPriority := statePriority[best.State]

	for i := 1; i < len(aliases); i++ {
		priority, ok := statePriority[aliases[i].State]
		if !ok {
			priority = 999
		}
		if priority < bestPriority {
			best = &aliases[i]
			bestPriority = priority
		}
	}

	return best
}
