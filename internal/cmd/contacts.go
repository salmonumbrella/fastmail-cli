package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/salmonumbrella/fastmail-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newContactsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Contacts management operations",
		Long: `Manage Fastmail contacts and address books.

Note: Fastmail may use CardDAV instead of JMAP for contacts.
If contacts are not available via JMAP, you'll receive an error.`,
	}

	cmd.AddCommand(newContactsListCmd(flags))
	cmd.AddCommand(newContactsGetCmd(flags))
	cmd.AddCommand(newContactsCreateCmd(flags))
	cmd.AddCommand(newContactsUpdateCmd(flags))
	cmd.AddCommand(newContactsDeleteCmd(flags))
	cmd.AddCommand(newContactsSearchCmd(flags))
	cmd.AddCommand(newContactsAddressBooksCmd(flags))

	return cmd
}

func newContactsListCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var addressbook string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contacts",
		Long: `List contacts from your address book.

Optionally filter by address book ID and limit the number of results.`,
		Example: `  fastmail contacts list
  fastmail contacts list --limit 50
  fastmail contacts list --addressbook <id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			contacts, err := client.GetContacts(cmd.Context(), addressbook, limit)
			if err != nil {
				return fmt.Errorf("failed to list contacts: %w", err)
			}

			// Sort by name
			sort.Slice(contacts, func(i, j int) bool {
				return contacts[i].Name < contacts[j].Name
			})

			if isJSON(cmd.Context()) {
				return printJSON(cmd, contacts)
			}

			if len(contacts) == 0 {
				outfmt.Errorf("No contacts found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(tw, "NAME\tEMAIL\tPHONE\tCOMPANY") //nolint:errcheck
			for _, contact := range contacts {
				email := "-"
				if len(contact.Emails) > 0 {
					email = contact.Emails[0].Value
				}
				phone := "-"
				if len(contact.Phones) > 0 {
					phone = contact.Phones[0].Value
				}
				company := contact.Company
				if company == "" {
					company = "-"
				}
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", //nolint:errcheck
					sanitizeTab(contact.Name),
					sanitizeTab(email),
					sanitizeTab(phone),
					sanitizeTab(company),
				)
			}
			_ = tw.Flush() //nolint:errcheck

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of contacts to retrieve")
	cmd.Flags().StringVar(&addressbook, "addressbook", "", "Filter by address book ID")

	return cmd
}

func newContactsGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <contactId>",
		Short: "Get a contact by ID",
		Long:  `Retrieve detailed information about a specific contact.`,
		Example: `  fastmail contacts get <id>
  fastmail contacts get <id> --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			contact, err := client.GetContactByID(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get contact: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, contact)
			}

			fmt.Printf("ID:       %s\n", contact.ID)
			fmt.Printf("Name:     %s\n", contact.Name)
			if contact.Company != "" {
				fmt.Printf("Company:  %s\n", contact.Company)
			}
			if contact.JobTitle != "" {
				fmt.Printf("Title:    %s\n", contact.JobTitle)
			}

			if len(contact.Emails) > 0 {
				fmt.Println("\nEmails:")
				for _, email := range contact.Emails {
					fmt.Printf("  [%s] %s\n", email.Type, email.Value)
				}
			}

			if len(contact.Phones) > 0 {
				fmt.Println("\nPhones:")
				for _, phone := range contact.Phones {
					fmt.Printf("  [%s] %s\n", phone.Type, phone.Value)
				}
			}

			if len(contact.Addresses) > 0 {
				fmt.Println("\nAddresses:")
				for _, addr := range contact.Addresses {
					fmt.Printf("  [%s]\n", addr.Type)
					if addr.Street != "" {
						fmt.Printf("    %s\n", addr.Street)
					}
					parts := []string{}
					if addr.City != "" {
						parts = append(parts, addr.City)
					}
					if addr.State != "" {
						parts = append(parts, addr.State)
					}
					if addr.PostalCode != "" {
						parts = append(parts, addr.PostalCode)
					}
					if len(parts) > 0 {
						fmt.Printf("    %s\n", strings.Join(parts, ", "))
					}
					if addr.Country != "" {
						fmt.Printf("    %s\n", addr.Country)
					}
				}
			}

			if contact.Birthday != "" {
				fmt.Printf("\nBirthday: %s\n", contact.Birthday)
			}
			if contact.Anniversary != "" {
				fmt.Printf("Anniversary: %s\n", contact.Anniversary)
			}

			if contact.Notes != "" {
				fmt.Printf("\nNotes:\n%s\n", contact.Notes)
			}

			fmt.Printf("\nUpdated: %s\n", contact.Updated.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	return cmd
}

func newContactsCreateCmd(flags *rootFlags) *cobra.Command {
	var name string
	var email string
	var phone string
	var company string
	var jobTitle string
	var notes string

	cmd := &cobra.Command{
		Use:   "create --name <name>",
		Short: "Create a new contact",
		Long: `Create a new contact with the specified details.

At minimum, you must provide a name. Other fields are optional.`,
		Example: `  fastmail contacts create --name "John Doe" --email "john@example.com"
  fastmail contacts create --name "Jane Smith" --email "jane@example.com" --phone "+1-555-1234" --company "Acme Corp"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("name is required")
			}

			client, err := getClient(flags)
			if err != nil {
				return err
			}

			contact := &jmap.Contact{
				Name:     name,
				Company:  company,
				JobTitle: jobTitle,
				Notes:    notes,
			}

			if email != "" {
				contact.Emails = []jmap.ContactEmail{
					{Type: "work", Value: email},
				}
			}

			if phone != "" {
				contact.Phones = []jmap.ContactPhone{
					{Type: "work", Value: phone},
				}
			}

			created, err := client.CreateContact(cmd.Context(), contact)
			if err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, created)
			}

			fmt.Printf("Created contact: %s (ID: %s)\n", created.Name, created.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Contact name (required)")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&company, "company", "", "Company name")
	cmd.Flags().StringVar(&jobTitle, "job-title", "", "Job title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")

	_ = cmd.MarkFlagRequired("name") //nolint:errcheck

	return cmd
}

func newContactsUpdateCmd(flags *rootFlags) *cobra.Command {
	var name string
	var email string
	var phone string
	var company string
	var jobTitle string
	var notes string

	cmd := &cobra.Command{
		Use:   "update <contactId>",
		Short: "Update a contact",
		Long: `Update an existing contact with new information.

Only the fields you specify will be updated.`,
		Example: `  fastmail contacts update <id> --name "John Doe Jr."
  fastmail contacts update <id> --email "newemail@example.com"
  fastmail contacts update <id> --company "New Corp" --job-title "CEO"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			updates := make(map[string]interface{})

			if name != "" {
				updates["name"] = name
			}
			if company != "" {
				updates["company"] = company
			}
			if jobTitle != "" {
				updates["jobTitle"] = jobTitle
			}
			if notes != "" {
				updates["notes"] = notes
			}
			if email != "" {
				updates["emails"] = []jmap.ContactEmail{
					{Type: "work", Value: email},
				}
			}
			if phone != "" {
				updates["phones"] = []jmap.ContactPhone{
					{Type: "work", Value: phone},
				}
			}

			if len(updates) == 0 {
				return fmt.Errorf("no updates specified")
			}

			updated, err := client.UpdateContact(cmd.Context(), args[0], updates)
			if err != nil {
				return fmt.Errorf("failed to update contact: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, updated)
			}

			fmt.Printf("Updated contact: %s\n", updated.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&company, "company", "", "Company name")
	cmd.Flags().StringVar(&jobTitle, "job-title", "", "Job title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")

	return cmd
}

func newContactsDeleteCmd(flags *rootFlags) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <contactId>",
		Short: "Delete a contact",
		Long:  `Delete a contact by ID. This action cannot be undone.`,
		Example: `  fastmail contacts delete <id>
  fastmail contacts delete <id> -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("Are you sure you want to delete this contact? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					outfmt.Errorf("Cancelled")
					return nil
				}
			}

			client, err := getClient(flags)
			if err != nil {
				return err
			}

			if err := client.DeleteContact(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("failed to delete contact: %w", err)
			}

			fmt.Println("Contact deleted")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func newContactsSearchCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search contacts",
		Long: `Search for contacts matching a query string.

The query is matched against contact names, emails, and other fields.`,
		Example: `  fastmail contacts search "john"
  fastmail contacts search "example.com"
  fastmail contacts search "acme" --limit 20`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			contacts, err := client.SearchContacts(cmd.Context(), args[0], limit)
			if err != nil {
				return fmt.Errorf("failed to search contacts: %w", err)
			}

			// Sort by name
			sort.Slice(contacts, func(i, j int) bool {
				return contacts[i].Name < contacts[j].Name
			})

			if isJSON(cmd.Context()) {
				return printJSON(cmd, contacts)
			}

			if len(contacts) == 0 {
				outfmt.Errorf("No contacts found matching '%s'", args[0])
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tEMAIL\tPHONE\tCOMPANY")
			for _, contact := range contacts {
				email := "-"
				if len(contact.Emails) > 0 {
					email = contact.Emails[0].Value
				}
				phone := "-"
				if len(contact.Phones) > 0 {
					phone = contact.Phones[0].Value
				}
				company := contact.Company
				if company == "" {
					company = "-"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					sanitizeTab(contact.Name),
					sanitizeTab(email),
					sanitizeTab(phone),
					sanitizeTab(company),
				)
			}
			tw.Flush()

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")

	return cmd
}

func newContactsAddressBooksCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addressbooks",
		Short: "List address books",
		Long:  `List all address books in your account.`,
		Example: `  fastmail contacts addressbooks
  fastmail contacts addressbooks --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(flags)
			if err != nil {
				return err
			}

			addressBooks, err := client.GetAddressBooks(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list address books: %w", err)
			}

			if isJSON(cmd.Context()) {
				return printJSON(cmd, addressBooks)
			}

			if len(addressBooks) == 0 {
				outfmt.Errorf("No address books found")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tDEFAULT\tSUBSCRIBED")
			for _, ab := range addressBooks {
				def := ""
				if ab.IsDefault {
					def = "yes"
				}
				sub := ""
				if ab.IsSubscribed {
					sub = "yes"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					ab.ID,
					sanitizeTab(ab.Name),
					def,
					sub,
				)
			}
			tw.Flush()

			return nil
		},
	}

	return cmd
}
