package cmd

import (
	"fmt"
	"io"
	"os"

	cerrors "github.com/salmonumbrella/fastmail-cli/internal/errors"
	"github.com/salmonumbrella/fastmail-cli/internal/format"
	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newEmailAttachmentsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attachments <emailId>",
		Aliases: []string{"att"},
		Short:   "List attachments for an email",
		Args:    cobra.ExactArgs(1),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			attachments, err := client.GetEmailAttachments(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get attachments: %w", err)
			}

			if app.IsJSON(cmd.Context()) {
				return app.PrintJSON(cmd, map[string]any{
					"emailId":     args[0],
					"attachments": attachments,
				})
			}

			if len(attachments) == 0 {
				printNoResults("No attachments")
				return nil
			}

			tw := outfmt.NewTabWriter()
			fmt.Fprintln(tw, "BLOB ID\tNAME\tTYPE\tSIZE")
			for _, att := range attachments {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					att.BlobID,
					outfmt.SanitizeTab(att.Name),
					att.Type,
					format.FormatBytes(att.Size),
				)
			}
			tw.Flush()

			return nil
		}),
	}

	return cmd
}

func newEmailDownloadCmd(app *App) *cobra.Command {
	var downloadAll bool
	var outputDir string

	cmd := &cobra.Command{
		Use:     "download <emailId> [blobId] [output-file]",
		Aliases: []string{"dl"},
		Short:   "Download email attachment(s)",
		Long: `Download email attachments.

With --all flag: Download all attachments from an email.
Without --all: Download a specific attachment by blob ID.

If output-file is not specified, attachments are saved with their original names.
Use --dir to specify the output directory (created if it doesn't exist).

Examples:
  # Download all attachments from an email to a directory
  fastmail email download ABC123 --all --dir ~/Downloads/attachments/

  # Download a specific attachment by blob ID
  fastmail email download ABC123 BLOB456

  # Download a specific attachment with custom output path
  fastmail email download ABC123 BLOB456 /tmp/my-file.pdf`,
		Args: cobra.RangeArgs(1, 3),
		RunE: runE(app, func(cmd *cobra.Command, args []string, app *App) error {
			client, err := app.JMAPClient()
			if err != nil {
				return err
			}

			emailID := args[0]

			// Handle --all flag: download all attachments
			if downloadAll {
				return downloadAllAttachments(cmd, client, app, emailID, outputDir)
			}

			// Single attachment download requires blobId
			if len(args) < 2 {
				return fmt.Errorf("blobId is required when not using --all flag\n\nUsage:\n  fastmail email download <emailId> <blobId> [output-file]\n  fastmail email download <emailId> --all [--dir <directory>]")
			}

			blobID := args[1]
			var outputFile string

			// Determine output file path
			if len(args) < 3 {
				var attachments []jmap.Attachment
				attachments, err = client.GetEmailAttachments(cmd.Context(), emailID)
				if err != nil {
					return fmt.Errorf("failed to get attachments: %w", err)
				}

				// Find the attachment with matching blob ID
				var found bool
				for _, att := range attachments {
					if att.BlobID == blobID {
						outputFile = att.Name
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("blob ID '%s' not found in email '%s'", blobID, emailID)
				}

				if outputFile == "" {
					outputFile = "attachment"
				}

				// SECURITY: Sanitize auto-detected filename to prevent path traversal attacks
				outputFile = format.SanitizeFilename(outputFile)

				// Apply output directory if specified
				if outputDir != "" {
					if err := os.MkdirAll(outputDir, 0o750); err != nil {
						return fmt.Errorf("failed to create directory '%s': %w", outputDir, err)
					}
					outputFile = fmt.Sprintf("%s/%s", outputDir, outputFile)
				}
			} else {
				outputFile = args[2]
			}

			return downloadSingleAttachment(cmd, client, app, emailID, blobID, outputFile)
		}),
	}

	cmd.Flags().BoolVarP(&downloadAll, "all", "a", false, "Download all attachments from the email")
	cmd.Flags().StringVarP(&outputDir, "dir", "d", "", "Output directory for downloaded files (created if it doesn't exist)")

	return cmd
}

// downloadAllAttachments downloads all attachments from an email
func downloadAllAttachments(cmd *cobra.Command, client jmap.EmailService, app *App, emailID, outputDir string) error {
	attachments, err := client.GetEmailAttachments(cmd.Context(), emailID)
	if err != nil {
		return fmt.Errorf("failed to get attachments: %w", err)
	}

	if len(attachments) == 0 {
		if app.IsJSON(cmd.Context()) {
			return app.PrintJSON(cmd, map[string]any{
				"emailId":     emailID,
				"attachments": []map[string]any{},
				"skipped":     []map[string]any{},
				"errors":      []map[string]any{},
				"total":       0,
			})
		}
		fmt.Println("No attachments to download")
		return nil
	}

	// Create output directory if specified
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", outputDir, err)
		}
	}

	var results []map[string]any
	var skipped []map[string]any
	var errors []map[string]any
	for _, att := range attachments {
		outputFile := format.SanitizeFilename(att.Name)
		if outputFile == "" {
			outputFile = fmt.Sprintf("attachment-%s", att.BlobID[:8])
		}

		if outputDir != "" {
			outputFile = fmt.Sprintf("%s/%s", outputDir, outputFile)
		}

		// Check if file exists
		if _, statErr := os.Stat(outputFile); statErr == nil {
			if app.IsJSON(cmd.Context()) {
				skipped = append(skipped, map[string]any{
					"blobId":     att.BlobID,
					"name":       att.Name,
					"outputFile": outputFile,
					"reason":     "already_exists",
				})
			} else {
				fmt.Printf("Skipping %s (already exists)\n", outputFile)
			}
			continue
		}

		// Download the blob
		reader, err := client.DownloadBlob(cmd.Context(), att.BlobID)
		if err != nil {
			if app.IsJSON(cmd.Context()) {
				errors = append(errors, map[string]any{
					"blobId": att.BlobID,
					"name":   att.Name,
					"step":   "download",
					"error":  err.Error(),
				})
			} else {
				fmt.Printf("Error downloading %s: %v\n", att.Name, err)
			}
			continue
		}

		// Create output file
		outFile, err := os.Create(outputFile)
		if err != nil {
			reader.Close()
			if app.IsJSON(cmd.Context()) {
				errors = append(errors, map[string]any{
					"blobId":     att.BlobID,
					"name":       att.Name,
					"outputFile": outputFile,
					"step":       "create_file",
					"error":      err.Error(),
				})
			} else {
				fmt.Printf("Error creating file %s: %v\n", outputFile, err)
			}
			continue
		}

		// Copy content
		written, err := io.Copy(outFile, reader)
		reader.Close()
		outFile.Close()

		if err != nil {
			if app.IsJSON(cmd.Context()) {
				errors = append(errors, map[string]any{
					"blobId":     att.BlobID,
					"name":       att.Name,
					"outputFile": outputFile,
					"step":       "write_file",
					"error":      err.Error(),
				})
			} else {
				fmt.Printf("Error writing %s: %v\n", outputFile, err)
			}
			continue
		}

		results = append(results, map[string]any{
			"blobId":     att.BlobID,
			"name":       att.Name,
			"outputFile": outputFile,
			"size":       written,
		})

		if !app.IsJSON(cmd.Context()) {
			fmt.Printf("Downloaded %s (%s)\n", outputFile, format.FormatBytes(written))
		}
	}

	if app.IsJSON(cmd.Context()) {
		return app.PrintJSON(cmd, map[string]any{
			"emailId":     emailID,
			"attachments": results,
			"skipped":     skipped,
			"errors":      errors,
			"total":       len(results),
		})
	}

	fmt.Printf("\nDownloaded %d attachment(s)\n", len(results))
	return nil
}

// downloadSingleAttachment downloads a single attachment by blob ID
func downloadSingleAttachment(cmd *cobra.Command, client jmap.EmailService, app *App, emailID, blobID, outputFile string) error {
	// Check if file already exists
	if _, statErr := os.Stat(outputFile); statErr == nil {
		return fmt.Errorf("file '%s' already exists. Specify a different output file", outputFile)
	}

	// Download the blob
	reader, err := client.DownloadBlob(cmd.Context(), blobID)
	if err != nil {
		return cerrors.WithContext(err, "downloading attachment")
	}
	defer reader.Close()

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy content
	written, err := io.Copy(outFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write attachment: %w", err)
	}

	if app.IsJSON(cmd.Context()) {
		return app.PrintJSON(cmd, map[string]any{
			"emailId":    emailID,
			"blobId":     blobID,
			"outputFile": outputFile,
			"size":       written,
		})
	}

	fmt.Printf("Downloaded attachment to %s (%s)\n", outputFile, format.FormatBytes(written))
	return nil
}
