package cmd

import "github.com/spf13/cobra"

func newEmailCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "email",
		Aliases: []string{"emails", "mail", "messages", "msg"},
		Short:   "Email operations",
	}

	cmd.AddCommand(newEmailListCmd(app))
	cmd.AddCommand(newEmailSearchCmd(app))
	cmd.AddCommand(newEmailGetCmd(app))
	cmd.AddCommand(newEmailSendCmd(app))
	cmd.AddCommand(newEmailForwardCmd(app))
	cmd.AddCommand(newEmailDeleteCmd(app))
	cmd.AddCommand(newEmailBulkDeleteCmd(app))
	cmd.AddCommand(newEmailMoveCmd(app))
	cmd.AddCommand(newEmailBulkMoveCmd(app))
	cmd.AddCommand(newEmailMarkReadCmd(app))
	cmd.AddCommand(newEmailBulkMarkReadCmd(app))
	cmd.AddCommand(newEmailThreadCmd(app))
	cmd.AddCommand(newEmailAttachmentsCmd(app))
	cmd.AddCommand(newEmailDownloadCmd(app))
	cmd.AddCommand(newEmailMailboxesCmd(app))
	cmd.AddCommand(newMailboxCreateCmd(app))
	cmd.AddCommand(newMailboxDeleteCmd(app))
	cmd.AddCommand(newMailboxRenameCmd(app))
	cmd.AddCommand(newEmailImportCmd(app))
	cmd.AddCommand(newEmailIdentitiesCmd(app))
	cmd.AddCommand(newIdentitySetDefaultCmd(app))
	cmd.AddCommand(newEmailTrackCmd(app))

	return cmd
}
