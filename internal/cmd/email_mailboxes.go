package cmd

import (
	"fmt"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newEmailMailboxesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mailboxes",
		Aliases: []string{"folders"},
		Short:   "List mailboxes (folders)",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			mailboxes, err := client.GetMailboxes(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get mailboxes: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, mailboxes)
			}

			tw := outfmt.NewTabWriter()
			fmt.Fprintln(tw, "ID\tNAME\tROLE\tUNREAD\tTOTAL")
			for _, mb := range mailboxes {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\n",
					mb.ID,
					outfmt.SanitizeTab(mb.Name),
					mb.Role,
					mb.UnreadEmails,
					mb.TotalEmails,
				)
			}
			tw.Flush()

			return nil
		}),
	}

	return cmd
}

func newMailboxCreateCmd(app *App) *cobra.Command {
	var parentID string

	cmd := &cobra.Command{
		Use:   "mailbox-create <name>",
		Short: "Create a new mailbox (folder)",
		Args:  cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			opts := jmap.CreateMailboxOpts{
				Name:     args[0],
				ParentID: parentID,
			}

			mailbox, err := client.CreateMailbox(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("failed to create mailbox: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, mailbox)
			}

			fmt.Printf("Created mailbox '%s' (ID: %s)\n", mailbox.Name, mailbox.ID)
			return nil
		}),
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent mailbox ID (for nested folders)")

	return cmd
}

func newMailboxDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailbox-delete <mailbox-id-or-name>",
		Short: "Delete a mailbox (folder)",
		Long:  "Delete a mailbox. Emails in the mailbox will be moved to trash.",
		Args:  cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// First resolve the mailbox ID
			mailboxID, err := client.ResolveMailboxID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("invalid mailbox: %w", err)
			}

			// Then get mailboxes to find the name for the prompt
			mailboxes, err := client.GetMailboxes(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get mailboxes: %w", err)
			}

			// Find the mailbox name for the confirmation prompt
			var mailboxName string
			for _, mb := range mailboxes {
				if mb.ID == mailboxID {
					mailboxName = mb.Name
					break
				}
			}

			// Prompt for confirmation unless --yes flag is set or JSON output mode
			confirmed, err := app.Confirm(cmd, false, fmt.Sprintf("Delete mailbox '%s' (ID: %s)? [y/N] ", mailboxName, mailboxID), "y", "yes")
			if err != nil {
				return err
			}
			if !confirmed {
				printCancelled()
				return nil
			}

			err = client.DeleteMailbox(cmd.Context(), mailboxID)
			if err != nil {
				return fmt.Errorf("failed to delete mailbox: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"status":  "deleted",
					"deleted": mailboxID,
				})
			}

			fmt.Printf("Deleted mailbox %s\n", mailboxID)
			return nil
		}),
	}

	return cmd
}

func newMailboxRenameCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailbox-rename <mailbox-id-or-name> <new-name>",
		Short: "Rename a mailbox (folder)",
		Args:  cobra.ExactArgs(2),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// Resolve mailbox ID or name
			mailboxID, err := client.ResolveMailboxID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("invalid mailbox: %w", err)
			}

			err = client.RenameMailbox(cmd.Context(), mailboxID, args[1])
			if err != nil {
				return fmt.Errorf("failed to rename mailbox: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"status":    "renamed",
					"mailboxId": mailboxID,
					"newName":   args[1],
				})
			}

			fmt.Printf("Renamed mailbox %s to '%s'\n", mailboxID, args[1])
			return nil
		}),
	}

	return cmd
}

func newEmailIdentitiesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identities",
		Short: "List sending identities (aliases)",
		Long:  "List all email identities/aliases you can send from.",
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			identities, err := client.GetIdentities(cmd.Context())
			if err != nil {
				return cerrors.WithContext(err, "fetching identities")
			}

			// Get the user-configured default identity
			accountEmail, accountErr := app.RequireAccount()
			if accountErr != nil {
				return accountErr
			}
			defaultIdentity, _ := config.GetDefaultIdentity(accountEmail)

			// Mark identities as default
			for i := range identities {
				if defaultIdentity != "" {
					// User has explicitly set a default
					identities[i].IsDefault = strings.EqualFold(identities[i].Email, defaultIdentity)
				} else {
					// Fall back to primary account identity (MayDelete=false)
					identities[i].IsDefault = !identities[i].MayDelete
				}
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, identities)
			}

			if len(identities) == 0 {
				printNoResults("No identities found")
				return nil
			}

			tw := outfmt.NewTabWriter()
			fmt.Fprintln(tw, "ID\tEMAIL\tNAME\tDEFAULT")
			for _, id := range identities {
				isDefaultStr := ""
				if id.IsDefault {
					isDefaultStr = "*"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					id.ID,
					id.Email,
					outfmt.SanitizeTab(id.Name),
					isDefaultStr,
				)
			}
			tw.Flush()

			return nil
		}),
	}

	return cmd
}

func newIdentitySetDefaultCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity-set-default <email>",
		Short: "Set the default sending identity",
		Long:  "Set which email identity to use by default when sending emails.",
		Args:  cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			targetEmail := strings.ToLower(strings.TrimSpace(args[0]))

			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			// Get all identities to validate the email exists
			identities, err := client.GetIdentities(cmd.Context())
			if err != nil {
				return cerrors.WithContext(err, "fetching identities")
			}

			// Validate the email is a valid identity
			var found bool
			var matchedIdentity jmap.Identity
			for _, id := range identities {
				if strings.EqualFold(id.Email, targetEmail) {
					found = true
					matchedIdentity = id
					break
				}
			}

			if !found {
				available := make([]string, len(identities))
				for i, id := range identities {
					available[i] = id.Email
				}
				return fmt.Errorf("identity %q not found; available identities: %s", targetEmail, strings.Join(available, ", "))
			}

			// Get the current account email
			accountEmail, err := app.RequireAccount()
			if err != nil {
				return err
			}

			// Save the default identity preference
			if err := config.SetDefaultIdentity(accountEmail, matchedIdentity.Email); err != nil {
				return fmt.Errorf("failed to save default identity: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"defaultIdentity": matchedIdentity.Email,
					"identityId":      matchedIdentity.ID,
				})
			}

			fmt.Printf("Default sending identity set to: %s\n", matchedIdentity.Email)
			return nil
		}),
	}

	return cmd
}
