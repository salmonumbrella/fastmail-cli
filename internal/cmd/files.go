package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/salmonumbrella/fastmail-cli/internal/config"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/salmonumbrella/fastmail-cli/internal/webdav"
	"github.com/spf13/cobra"
)

func newFilesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "File storage operations",
		Long: `Manage files in Fastmail file storage using WebDAV.

Files are stored at https://myfiles.fastmail.com/ and can be accessed
via WebDAV protocol for upload, download, and management.`,
	}

	cmd.AddCommand(newFilesListCmd(flags))
	cmd.AddCommand(newFilesUploadCmd(flags))
	cmd.AddCommand(newFilesDownloadCmd(flags))
	cmd.AddCommand(newFilesMkdirCmd(flags))
	cmd.AddCommand(newFilesDeleteCmd(flags))
	cmd.AddCommand(newFilesMoveCmd(flags))

	return cmd
}

func newFilesListCmd(flags *rootFlags) *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "list [path]",
		Short: "List files and directories",
		Long: `List files and directories in Fastmail file storage.

Without a path argument, lists files in the root directory.
Use --recursive to list all files recursively.`,
		Example: `  fastmail files list
  fastmail files list /Documents
  fastmail files list /Documents --recursive`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getWebDAVClient(flags)
			if err != nil {
				return err
			}

			path := "/"
			if len(args) > 0 {
				path = args[0]
			}

			if recursive {
				return listRecursive(cmd, client, path)
			}

			files, err := client.List(cmd.Context(), path)
			if err != nil {
				return fmt.Errorf("failed to list files: %w", err)
			}

			// Sort by name
			sort.Slice(files, func(i, j int) bool {
				// Directories first, then files
				if files[i].IsDirectory != files[j].IsDirectory {
					return files[i].IsDirectory
				}
				return files[i].Name < files[j].Name
			})

			if isJSON(cmd.Context()) {
				return printJSON(cmd, files)
			}

			if len(files) == 0 {
				outfmt.Errorf("No files found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tTYPE\tSIZE\tMODIFIED")
			for _, file := range files {
				fileType := "file"
				if file.IsDirectory {
					fileType = "dir"
				}

				size := formatSize(file.Size)
				if file.IsDirectory {
					size = "-"
				}

				modified := file.LastModified.Format("2006-01-02 15:04")

				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					file.Name,
					fileType,
					size,
					modified,
				)
			}
			tw.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "List files recursively")

	return cmd
}

func newFilesUploadCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <local-file> [remote-path]",
		Short: "Upload a file to file storage",
		Long: `Upload a local file to Fastmail file storage.

If remote-path is not specified, the file is uploaded to the root directory
with the same name as the local file.`,
		Example: `  fastmail files upload document.pdf
  fastmail files upload document.pdf /Documents/report.pdf
  fastmail files upload ~/Downloads/photo.jpg /Photos/`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getWebDAVClient(flags)
			if err != nil {
				return err
			}

			localPath := args[0]
			remotePath := "/"

			if len(args) > 1 {
				remotePath = args[1]
			}

			// If remote path is a directory (ends with /), append the local filename
			if strings.HasSuffix(remotePath, "/") {
				remotePath = path.Join(remotePath, filepath.Base(localPath))
			}

			// Verify local file exists
			if _, err := os.Stat(localPath); err != nil {
				return fmt.Errorf("local file not found: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Uploading %s to %s...\n", localPath, remotePath)

			if err := client.Upload(cmd.Context(), localPath, remotePath); err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}

			fmt.Fprintln(os.Stderr, "File uploaded successfully")

			return nil
		},
	}

	return cmd
}

func newFilesDownloadCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <remote-path> [local-file]",
		Short: "Download a file from file storage",
		Long: `Download a file from Fastmail file storage to a local path.

If local-file is not specified, the file is downloaded to the current
directory with the same name as the remote file.`,
		Example: `  fastmail files download /Documents/report.pdf
  fastmail files download /Documents/report.pdf ~/Downloads/report.pdf
  fastmail files download /Photos/image.jpg .`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getWebDAVClient(flags)
			if err != nil {
				return err
			}

			remotePath := args[0]
			localPath := filepath.Base(remotePath)

			if len(args) > 1 {
				localPath = args[1]
			}

			// If local path is a directory, append the remote filename
			if info, err := os.Stat(localPath); err == nil && info.IsDir() {
				localPath = filepath.Join(localPath, filepath.Base(remotePath))
			}

			fmt.Fprintf(os.Stderr, "Downloading %s to %s...\n", remotePath, localPath)

			if err := client.Download(cmd.Context(), remotePath, localPath); err != nil {
				return fmt.Errorf("download failed: %w", err)
			}

			fmt.Fprintln(os.Stderr, "File downloaded successfully")

			return nil
		},
	}

	return cmd
}

func newFilesMkdirCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mkdir <path>",
		Short: "Create a directory",
		Long:  `Create a new directory in Fastmail file storage.`,
		Example: `  fastmail files mkdir /Documents
  fastmail files mkdir /Photos/Vacation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getWebDAVClient(flags)
			if err != nil {
				return err
			}

			dirPath := args[0]

			if err := client.Mkdir(cmd.Context(), dirPath); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Directory created: %s\n", dirPath)

			return nil
		},
	}

	return cmd
}

func newFilesDeleteCmd(flags *rootFlags) *cobra.Command {
	var skipConfirmation bool

	cmd := &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete a file or directory",
		Long: `Delete a file or directory from Fastmail file storage.

WARNING: This operation cannot be undone. Use with caution.`,
		Example: `  fastmail files delete /Documents/old.pdf
  fastmail files delete /OldFolder -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getWebDAVClient(flags)
			if err != nil {
				return err
			}

			path := args[0]

			// Confirm deletion unless -y flag is provided
			if !skipConfirmation {
				fmt.Printf("Are you sure you want to delete %s? (yes/no): ", path)
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "yes" && response != "y" {
					fmt.Fprintln(os.Stderr, "Delete cancelled")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), path); err != nil {
				return fmt.Errorf("delete failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Deleted: %s\n", path)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func newFilesMoveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <source> <destination>",
		Short: "Move or rename a file or directory",
		Long: `Move or rename a file or directory in Fastmail file storage.

This operation will fail if the destination already exists.`,
		Example: `  fastmail files move /old.pdf /new.pdf
  fastmail files move /Documents/report.pdf /Archive/
  fastmail files move /OldFolder /NewFolder`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getWebDAVClient(flags)
			if err != nil {
				return err
			}

			source := args[0]
			destination := args[1]

			if err := client.Move(cmd.Context(), source, destination); err != nil {
				return fmt.Errorf("move failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Moved: %s -> %s\n", source, destination)

			return nil
		},
	}

	return cmd
}

// getWebDAVClient creates a WebDAV client using the configured account token
func getWebDAVClient(flags *rootFlags) (*webdav.Client, error) {
	account, err := requireAccount(flags)
	if err != nil {
		return nil, err
	}

	token, err := config.GetToken(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for %s: %w", account, err)
	}

	return webdav.NewClient(token), nil
}

// listRecursive lists files recursively
func listRecursive(cmd *cobra.Command, client *webdav.Client, rootPath string) error {
	type pathInfo struct {
		path  string
		depth int
	}

	ctx := cmd.Context()
	visited := make(map[string]bool)
	queue := []pathInfo{{path: rootPath, depth: 0}}
	var allFiles []struct {
		file  webdav.FileInfo
		depth int
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Skip if already visited (avoid infinite loops)
		if visited[current.path] {
			continue
		}
		visited[current.path] = true

		files, err := client.List(ctx, current.path)
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", current.path, err)
		}

		for _, file := range files {
			allFiles = append(allFiles, struct {
				file  webdav.FileInfo
				depth int
			}{file: file, depth: current.depth})

			// Add directories to queue for recursive listing
			if file.IsDirectory {
				queue = append(queue, pathInfo{
					path:  file.Path,
					depth: current.depth + 1,
				})
			}
		}
	}

	if isJSON(ctx) {
		// For JSON output, just output the files
		files := make([]webdav.FileInfo, len(allFiles))
		for i, f := range allFiles {
			files[i] = f.file
		}
		return printJSON(cmd, files)
	}

	if len(allFiles) == 0 {
		outfmt.Errorf("No files found")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "PATH\tTYPE\tSIZE\tMODIFIED")
	for _, item := range allFiles {
		file := item.file
		fileType := "file"
		if file.IsDirectory {
			fileType = "dir"
		}

		size := formatSize(file.Size)
		if file.IsDirectory {
			size = "-"
		}

		modified := file.LastModified.Format("2006-01-02 15:04")

		// Add indentation based on depth
		indent := strings.Repeat("  ", item.depth)
		path := indent + file.Name

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			path,
			fileType,
			size,
			modified,
		)
	}
	tw.Flush()

	return nil
}
