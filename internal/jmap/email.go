package jmap

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/salmonumbrella/fastmail-cli/internal/logging"
)

// Mailbox represents a JMAP mailbox.
type Mailbox struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role,omitempty"`
	TotalEmails   int    `json:"totalEmails"`
	UnreadEmails  int    `json:"unreadEmails"`
	TotalThreads  int    `json:"totalThreads,omitempty"`
	UnreadThreads int    `json:"unreadThreads,omitempty"`
}

// EmailAddress represents an email address with optional name.
type EmailAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// Email represents a JMAP email.
type Email struct {
	ID            string               `json:"id"`
	ThreadID      string               `json:"threadId,omitempty"`
	Subject       string               `json:"subject"`
	From          []EmailAddress       `json:"from"`
	To            []EmailAddress       `json:"to"`
	CC            []EmailAddress       `json:"cc"`
	BCC           []EmailAddress       `json:"bcc"`
	ReplyTo       []EmailAddress       `json:"replyTo"`
	ReceivedAt    string               `json:"receivedAt"`
	Preview       string               `json:"preview,omitempty"`
	HasAttachment bool                 `json:"hasAttachment"`
	Keywords      map[string]bool      `json:"keywords,omitempty"`
	MailboxIDs    map[string]bool      `json:"mailboxIds,omitempty"`
	BodyValues    map[string]BodyValue `json:"bodyValues,omitempty"`
	TextBody      []BodyPart           `json:"textBody"`
	HTMLBody      []BodyPart           `json:"htmlBody"`
	Attachments   []Attachment         `json:"attachments"`
	// Headers for threading replies
	MessageID  []string `json:"messageId"`
	InReplyTo  []string `json:"inReplyTo"`
	References []string `json:"references"`
}

// BodyValue represents email body content.
type BodyValue struct {
	Value string `json:"value"`
}

// BodyPart represents a body part reference.
type BodyPart struct {
	PartID string `json:"partId"`
	Type   string `json:"type"`
}

// Attachment represents an email attachment.
type Attachment struct {
	PartID string `json:"partId"`
	BlobID string `json:"blobId"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int64  `json:"size"`
}

// Identity represents a sending identity.
type Identity struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email"`
	MayDelete bool   `json:"mayDelete"`
	IsDefault bool   `json:"isDefault,omitempty"` // CLI preference, not JMAP property
}

// AttachmentOpts represents an attachment to include when sending an email.
type AttachmentOpts struct {
	BlobID string // Required: blob ID from UploadBlob
	Name   string // Required: filename to display
	Type   string // Required: MIME type (e.g., "application/pdf")
}

// SendEmailOpts contains options for sending an email.
type SendEmailOpts struct {
	To        []string
	CC        []string
	BCC       []string
	Subject   string
	TextBody  string
	HTMLBody  string
	From      string
	MailboxID string
	// For replies - set these to thread the email properly
	InReplyTo  []string
	References []string
	// Attachments to include (requires uploading blobs first via UploadBlob)
	Attachments []AttachmentOpts
}

// GetMailboxes retrieves all mailboxes for the account.
func (c *Client) GetMailboxes(ctx context.Context) ([]Mailbox, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Mailbox/get", map[string]any{"accountId": session.AccountID}, "mailboxes"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Extract mailboxes from response
	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	list, ok := result["list"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected list format")
	}

	mailboxes := make([]Mailbox, 0, len(list))
	for _, item := range list {
		mb, ok := item.(map[string]any)
		if !ok {
			continue
		}

		mailbox := Mailbox{
			ID:            getString(mb, "id"),
			Name:          getString(mb, "name"),
			Role:          getString(mb, "role"),
			TotalEmails:   getInt(mb, "totalEmails"),
			UnreadEmails:  getInt(mb, "unreadEmails"),
			TotalThreads:  getInt(mb, "totalThreads"),
			UnreadThreads: getInt(mb, "unreadThreads"),
		}
		mailboxes = append(mailboxes, mailbox)
	}

	return mailboxes, nil
}

// GetMailboxByName finds a mailbox by name (case-insensitive).
// Returns ErrMailboxNotFound if no mailbox matches the given name or role.
func (c *Client) GetMailboxByName(ctx context.Context, name string) (*Mailbox, error) {
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for i := range mailboxes {
		if strings.ToLower(mailboxes[i].Name) == nameLower {
			return &mailboxes[i], nil
		}
		// Also check role (e.g., "inbox", "sent", "drafts", "trash")
		if strings.ToLower(mailboxes[i].Role) == nameLower {
			return &mailboxes[i], nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrMailboxNotFound, name)
}

// ResolveMailboxID takes either a mailbox ID or name and returns the ID.
// It first tries to match by name/role, then validates if it's a valid mailbox ID.
// Returns ErrMailboxNotFound if the identifier doesn't match any mailbox.
func (c *Client) ResolveMailboxID(ctx context.Context, idOrName string) (string, error) {
	if idOrName == "" {
		return "", fmt.Errorf("mailbox identifier cannot be empty")
	}

	// Try name/role lookup first
	mb, err := c.GetMailboxByName(ctx, idOrName)
	if err == nil {
		return mb.ID, nil
	}

	// If it wasn't a name, check if it's a valid ID
	if !errors.Is(err, ErrMailboxNotFound) {
		return "", err // Some other error (network, etc.)
	}

	// Verify it's a valid mailbox ID
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching mailboxes: %w", err)
	}

	for _, mb := range mailboxes {
		if mb.ID == idOrName {
			return idOrName, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrMailboxNotFound, idOrName)
}

// GetEmails retrieves emails from a mailbox.
func (c *Client) GetEmails(ctx context.Context, mailboxID string, limit int) ([]Email, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	filter := map[string]any{}
	if mailboxID != "" {
		filter["inMailbox"] = mailboxID
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/query", map[string]any{
				"accountId": session.AccountID,
				"filter":    filter,
				"sort":      []map[string]any{{"property": "receivedAt", "isAscending": false}},
				"limit":     limit,
			}, "query"},
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"#ids":       map[string]any{"resultOf": "query", "name": "Email/query", "path": "/ids"},
				"properties": []string{"id", "subject", "from", "to", "receivedAt", "preview", "hasAttachment", "keywords", "threadId"},
			}, "emails"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return parseEmailList(resp.MethodResponses[1])
}

// GetEmailByID retrieves a specific email by ID.
func (c *Client) GetEmailByID(ctx context.Context, id string) (*Email, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/get", map[string]any{
				"accountId": session.AccountID,
				"ids":       []string{id},
				"properties": []string{
					"id", "subject", "from", "to", "cc", "bcc", "replyTo", "receivedAt",
					"textBody", "htmlBody", "attachments", "bodyValues", "keywords", "threadId",
					"messageId", "inReplyTo", "references",
				},
				"bodyProperties":      []string{"partId", "blobId", "type", "size"},
				"fetchTextBodyValues": true,
				"fetchHTMLBodyValues": true,
			}, "email"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Check for notFound
	if notFound, notFoundOK := result["notFound"].([]any); notFoundOK && len(notFound) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrEmailNotFound, id)
	}

	list, ok := result["list"].([]any)
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("email with ID '%s' not found or not accessible", id)
	}

	emailData, ok := list[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected email format")
	}

	return parseEmail(emailData), nil
}

// EmailSearchFilter contains JMAP filter options for email search.
type EmailSearchFilter struct {
	Text   string // Full-text search query
	After  string // RFC3339 timestamp - emails received after this time
	Before string // RFC3339 timestamp - emails received before this time
}

// ToJMAPFilter converts the EmailSearchFilter to a JMAP filter map.
func (f *EmailSearchFilter) ToJMAPFilter() map[string]any {
	filter := map[string]any{}
	if f.Text != "" {
		filter["text"] = f.Text
	}
	if f.After != "" {
		filter["after"] = f.After
	}
	if f.Before != "" {
		filter["before"] = f.Before
	}
	return filter
}

// SearchEmails searches for emails matching a filter.
func (c *Client) SearchEmails(ctx context.Context, searchFilter *EmailSearchFilter, limit int) ([]Email, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	filter := map[string]any{}
	if searchFilter != nil {
		filter = searchFilter.ToJMAPFilter()
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/query", map[string]any{
				"accountId": session.AccountID,
				"filter":    filter,
				"sort":      []map[string]any{{"property": "receivedAt", "isAscending": false}},
				"limit":     limit,
			}, "query"},
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"#ids":       map[string]any{"resultOf": "query", "name": "Email/query", "path": "/ids"},
				"properties": []string{"id", "subject", "from", "to", "cc", "receivedAt", "preview", "hasAttachment", "keywords", "threadId"},
			}, "emails"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return parseEmailList(resp.MethodResponses[1])
}

// GetDrafts retrieves all draft emails.
func (c *Client) GetDrafts(ctx context.Context, limit int) ([]Email, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Find drafts mailbox
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return nil, err
	}

	var draftsMailboxID string
	for _, mb := range mailboxes {
		if mb.Role == "drafts" {
			draftsMailboxID = mb.ID
			break
		}
	}
	if draftsMailboxID == "" {
		return nil, ErrNoDraftsMailbox
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/query", map[string]any{
				"accountId": session.AccountID,
				"filter": map[string]any{
					"inMailbox":  draftsMailboxID,
					"hasKeyword": "$draft",
				},
				"sort":  []map[string]any{{"property": "receivedAt", "isAscending": false}},
				"limit": limit,
			}, "query"},
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"#ids":       map[string]any{"resultOf": "query", "name": "Email/query", "path": "/ids"},
				"properties": []string{"id", "subject", "from", "to", "cc", "receivedAt", "preview", "hasAttachment", "keywords", "threadId"},
			}, "emails"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return parseEmailList(resp.MethodResponses[1])
}

// UpdateDraft updates an existing draft email.
func (c *Client) UpdateDraft(ctx context.Context, draftID string, opts SendEmailOpts) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	// Build email update object
	update := map[string]any{}

	if opts.From != "" {
		update["from"] = []map[string]string{{"email": opts.From}}
	}

	if len(opts.To) > 0 {
		to := make([]map[string]string, len(opts.To))
		for i, addr := range opts.To {
			to[i] = map[string]string{"email": addr}
		}
		update["to"] = to
	}

	if len(opts.CC) > 0 {
		cc := make([]map[string]string, len(opts.CC))
		for i, addr := range opts.CC {
			cc[i] = map[string]string{"email": addr}
		}
		update["cc"] = cc
	}

	if len(opts.BCC) > 0 {
		bcc := make([]map[string]string, len(opts.BCC))
		for i, addr := range opts.BCC {
			bcc[i] = map[string]string{"email": addr}
		}
		update["bcc"] = bcc
	}

	if opts.Subject != "" {
		update["subject"] = opts.Subject
	}

	// Body updates require setting both bodyValues and text/html body parts.
	if opts.TextBody != "" || opts.HTMLBody != "" {
		bodyValues := map[string]map[string]string{}
		if opts.TextBody != "" {
			update["textBody"] = []map[string]string{{"partId": "text", "type": "text/plain"}}
			bodyValues["text"] = map[string]string{"value": opts.TextBody}
		}
		if opts.HTMLBody != "" {
			update["htmlBody"] = []map[string]string{{"partId": "html", "type": "text/html"}}
			bodyValues["html"] = map[string]string{"value": opts.HTMLBody}
		}
		update["bodyValues"] = bodyValues
	}

	if len(update) == 0 {
		return nil
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					draftID: update,
				},
			}, "update"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		if errInfo, exists := notUpdated[draftID]; exists {
			return fmt.Errorf("failed to update draft: %v", errInfo)
		}
	}

	return nil
}

// SendDraft sends an existing draft email.
// Uses JMAP's onSuccessUpdateEmail to atomically update the email only if submission succeeds.
// If submission fails, the email stays in Drafts with $draft keyword intact.
func (c *Client) SendDraft(ctx context.Context, draftID string) (string, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return "", err
	}

	// Get the draft to verify it's a draft
	draft, err := c.GetEmailByID(ctx, draftID)
	if err != nil {
		return "", fmt.Errorf("failed to get draft: %w", err)
	}
	if draft.Keywords != nil && !draft.Keywords["$draft"] {
		return "", fmt.Errorf("email %s is not a draft", draftID)
	}

	// Get default identity
	identity, err := c.getDefaultIdentity(ctx)
	if err != nil {
		return "", err
	}

	// Find sent mailbox
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return "", err
	}
	var sentMailboxID string
	for _, mb := range mailboxes {
		if mb.Role == "sent" {
			sentMailboxID = mb.ID
			break
		}
	}
	if sentMailboxID == "" {
		return "", ErrNoSentMailbox
	}

	// Use onSuccessUpdateEmail to atomically update the email only if submission succeeds.
	// The key "#send" references the creation ID "send" in the create map.
	// If submission fails, the email stays unchanged in Drafts.
	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail", "urn:ietf:params:jmap:submission"},
		MethodCalls: []MethodCall{
			{"EmailSubmission/set", map[string]any{
				"accountId": session.AccountID,
				"create": map[string]any{
					"send": map[string]any{
						"identityId": identity.ID,
						"emailId":    draftID,
					},
				},
				// onSuccessUpdateEmail: updates to apply to emails only if submission succeeds
				// The key "#send" is a back-reference to the emailId of the "send" submission
				"onSuccessUpdateEmail": map[string]any{
					"#send": map[string]any{
						"keywords/$draft": nil, // Remove draft keyword
						"mailboxIds":      map[string]bool{sentMailboxID: true},
					},
				},
			}, "send"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	// Check submission result (now it's the first/only response)
	if len(resp.MethodResponses) >= 1 {
		if result, ok := resp.MethodResponses[0][1].(map[string]any); ok {
			if created, ok := result["created"].(map[string]any); ok {
				if send, ok := created["send"].(map[string]any); ok {
					if id, ok := send["id"].(string); ok {
						return id, nil
					}
				}
			}
			if notCreated, ok := result["notCreated"].(map[string]any); ok {
				if errInfo, exists := notCreated["send"]; exists {
					return "", fmt.Errorf("failed to send draft: %v", errInfo)
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected submission response: no created or notCreated field")
}

// CreateReplyDraft creates a draft that is threaded as a reply to an existing email.
func (c *Client) CreateReplyDraft(ctx context.Context, replyToID string, opts SendEmailOpts) (string, error) {
	// Fetch the original email to get threading headers
	original, err := c.GetEmailByID(ctx, replyToID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch original email: %w", err)
	}

	// Set up threading: InReplyTo = original's MessageID
	if len(original.MessageID) > 0 {
		opts.InReplyTo = original.MessageID
	}

	// References = original's References + original's MessageID
	refs := make([]string, 0)
	if len(original.References) > 0 {
		refs = append(refs, original.References...)
	}
	if len(original.MessageID) > 0 {
		refs = append(refs, original.MessageID...)
	}
	if len(refs) > 0 {
		opts.References = refs
	}

	// If no To specified, reply to sender (use ReplyTo if available, else From)
	if len(opts.To) == 0 {
		if len(original.ReplyTo) > 0 {
			for _, addr := range original.ReplyTo {
				opts.To = append(opts.To, addr.Email)
			}
		} else if len(original.From) > 0 {
			for _, addr := range original.From {
				opts.To = append(opts.To, addr.Email)
			}
		}
	}

	// If no subject, add "Re: " prefix
	if opts.Subject == "" && original.Subject != "" {
		subj := original.Subject
		if !strings.HasPrefix(strings.ToLower(subj), "re:") {
			subj = "Re: " + subj
		}
		opts.Subject = subj
	}

	// If no From specified, check if the original email was sent to a masked email
	// and use that as the reply address to maintain identity consistency.
	// Masked emails can be used for sending; the client will create a temporary
	// sending identity if needed.
	if opts.From == "" {
		maskedFrom := c.findMaskedEmailRecipient(ctx, original)
		if maskedFrom != "" {
			opts.From = maskedFrom
		}
	}

	return c.SaveDraft(ctx, opts)
}

// findMaskedEmailRecipient checks if any recipient address in the email is a masked email.
// Returns the masked email address if found, empty string otherwise.
func (c *Client) findMaskedEmailRecipient(ctx context.Context, email *Email) string {
	// Get all masked emails for this account
	maskedEmails, err := c.GetMaskedEmails(ctx)
	if err != nil {
		// If we can't fetch masked emails, just continue without setting From
		return ""
	}

	// Build a set of masked email addresses for quick lookup
	maskedSet := make(map[string]bool)
	for _, me := range maskedEmails {
		if me.State == MaskedEmailEnabled || me.State == MaskedEmailPending {
			maskedSet[strings.ToLower(me.Email)] = true
		}
	}

	// Check To recipients first
	for _, addr := range email.To {
		if maskedSet[strings.ToLower(addr.Email)] {
			return addr.Email
		}
	}

	// Check CC recipients
	for _, addr := range email.CC {
		if maskedSet[strings.ToLower(addr.Email)] {
			return addr.Email
		}
	}

	return ""
}

// SaveDraft saves an email as a draft without sending it.
func (c *Client) SaveDraft(ctx context.Context, opts SendEmailOpts) (string, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return "", err
	}

	// Determine the from address
	// For drafts, we can use any address (identity validation only applies to sending)
	fromEmail := opts.From
	if fromEmail == "" {
		// No from specified, use default identity
		var identities []Identity
		identities, err = c.GetIdentities(ctx)
		if err != nil {
			return "", err
		}
		if len(identities) == 0 {
			return "", ErrNoIdentities
		}
		// Use primary (non-deletable) identity as default
		for _, id := range identities {
			if !id.MayDelete {
				fromEmail = id.Email
				break
			}
		}
		if fromEmail == "" {
			fromEmail = identities[0].Email
		}
	}

	// Get drafts mailbox
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return "", err
	}

	var draftsMailbox *Mailbox
	for i := range mailboxes {
		if mailboxes[i].Role == "drafts" {
			draftsMailbox = &mailboxes[i]
			break
		}
	}

	if draftsMailbox == nil {
		return "", ErrNoDraftsMailbox
	}

	// Build email object
	emailObj := map[string]any{
		"mailboxIds": map[string]bool{draftsMailbox.ID: true},
		"keywords":   map[string]bool{"$draft": true},
		"from":       []map[string]string{{"email": fromEmail}},
		"subject":    opts.Subject,
	}

	// Add recipients
	if len(opts.To) > 0 {
		to := make([]map[string]string, len(opts.To))
		for i, addr := range opts.To {
			to[i] = map[string]string{"email": addr}
		}
		emailObj["to"] = to
	}

	if len(opts.CC) > 0 {
		cc := make([]map[string]string, len(opts.CC))
		for i, addr := range opts.CC {
			cc[i] = map[string]string{"email": addr}
		}
		emailObj["cc"] = cc
	}

	if len(opts.BCC) > 0 {
		bcc := make([]map[string]string, len(opts.BCC))
		for i, addr := range opts.BCC {
			bcc[i] = map[string]string{"email": addr}
		}
		emailObj["bcc"] = bcc
	}

	// Add body
	bodyValues := make(map[string]map[string]string)
	if opts.TextBody != "" {
		emailObj["textBody"] = []map[string]string{{"partId": "text", "type": "text/plain"}}
		bodyValues["text"] = map[string]string{"value": opts.TextBody}
	}
	if opts.HTMLBody != "" {
		emailObj["htmlBody"] = []map[string]string{{"partId": "html", "type": "text/html"}}
		bodyValues["html"] = map[string]string{"value": opts.HTMLBody}
	}
	emailObj["bodyValues"] = bodyValues

	// Add attachments if provided
	if len(opts.Attachments) > 0 {
		attachments := make([]map[string]any, len(opts.Attachments))
		for i, att := range opts.Attachments {
			attachments[i] = map[string]any{
				"blobId":      att.BlobID,
				"name":        att.Name,
				"type":        att.Type,
				"disposition": "attachment",
			}
		}
		emailObj["attachments"] = attachments
	}

	// Add threading headers for replies
	if len(opts.InReplyTo) > 0 {
		emailObj["inReplyTo"] = opts.InReplyTo
	}
	if len(opts.References) > 0 {
		emailObj["references"] = opts.References
	}

	// Create draft (no EmailSubmission - just save)
	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"create":    map[string]any{"draft": emailObj},
			}, "createDraft"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	// Check email creation
	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	if notCreated, ok := result["notCreated"].(map[string]any); ok {
		if errInfo, exists := notCreated["draft"]; exists {
			return "", fmt.Errorf("failed to create draft: %v", errInfo)
		}
	}

	// Extract draft ID
	if created, ok := result["created"].(map[string]any); ok {
		if draft, ok := created["draft"].(map[string]any); ok {
			if id, ok := draft["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("draft created but ID not returned")
}

// SendEmail sends an email.
func (c *Client) SendEmail(ctx context.Context, opts SendEmailOpts) (string, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return "", err
	}

	// Get identities for authorization
	identities, err := c.GetIdentities(ctx)
	if err != nil {
		return "", err
	}

	if len(identities) == 0 {
		return "", ErrNoIdentities
	}

	// Find default identity (for authorization when sending from masked email)
	var defaultIdentity *Identity
	for i := range identities {
		if !identities[i].MayDelete {
			defaultIdentity = &identities[i]
			break
		}
	}
	if defaultIdentity == nil {
		defaultIdentity = &identities[0]
	}

	// Determine authorization identity and sending addresses
	// - authIdentityID: used for identityId (JMAP authorization)
	// - authIdentityEmail: used for Sender header when it differs from From
	// - sendFromEmail: used in email From header (what recipient sees, can be masked email)
	// - envelopeFromEmail: used in SMTP envelope mailFrom (must be verified identity, or empty to omit envelope)
	// - isMaskedEmail: true if sending from a masked email (requires special handling)
	var authIdentityID string
	var authIdentityEmail string
	var sendFromEmail string
	var envelopeFromEmail string
	var isMaskedEmail bool
	var tempIdentityID string

	if opts.From != "" {
		// Check if From matches an identity
		for i := range identities {
			if strings.EqualFold(identities[i].Email, opts.From) {
				authIdentityID = identities[i].ID
				authIdentityEmail = identities[i].Email
				sendFromEmail = identities[i].Email
				envelopeFromEmail = identities[i].Email
				break
			}
		}

		// If not an identity, check if it's a masked email
		if authIdentityID == "" {
			maskedEmails, maskedErr := c.GetMaskedEmails(ctx)
			if maskedErr == nil {
				for _, me := range maskedEmails {
					if strings.EqualFold(me.Email, opts.From) && (me.State == MaskedEmailEnabled || me.State == MaskedEmailPending) {
						isMaskedEmail = true
						sendFromEmail = me.Email
						envelopeFromEmail = "" // Let Fastmail derive the envelope for masked emails.

						// Create a temporary identity for this masked email when possible.
						maskedID, identityErr := c.createIdentity(ctx, me.Email)
						if identityErr == nil && maskedID != "" {
							authIdentityID = maskedID
							authIdentityEmail = me.Email
							tempIdentityID = maskedID
						} else {
							// Fallback: Use default identity for authorization,
							// and the masked email for From header.
							authIdentityID = defaultIdentity.ID
							authIdentityEmail = defaultIdentity.Email
						}
						break
					}
				}
			}
		}

		// If neither identity nor masked email, error
		if authIdentityID == "" {
			availableIdentities := make([]string, len(identities))
			for i, id := range identities {
				availableIdentities[i] = id.Email
			}
			return "", &InvalidFromAddressError{
				AttemptedAddress:    opts.From,
				AvailableIdentities: availableIdentities,
				IsMaskedEmail:       false,
			}
		}
	} else {
		// No From specified, use default identity
		authIdentityID = defaultIdentity.ID
		authIdentityEmail = defaultIdentity.Email
		sendFromEmail = defaultIdentity.Email
		envelopeFromEmail = defaultIdentity.Email
	}

	if tempIdentityID != "" {
		defer func() {
			if delErr := c.deleteIdentity(ctx, tempIdentityID); delErr != nil {
				logging.FromContext(ctx).Debug("failed to delete temporary identity", "identityID", tempIdentityID, "error", delErr)
			}
		}()
	}

	// Get mailboxes
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return "", err
	}

	var draftsMailbox, sentMailbox *Mailbox
	for i := range mailboxes {
		if mailboxes[i].Role == "drafts" {
			draftsMailbox = &mailboxes[i]
		}
		if mailboxes[i].Role == "sent" {
			sentMailbox = &mailboxes[i]
		}
	}

	if draftsMailbox == nil {
		return "", ErrNoDraftsMailbox
	}
	if sentMailbox == nil {
		return "", ErrNoSentMailbox
	}

	// Ensure we have at least one body type
	if opts.TextBody == "" && opts.HTMLBody == "" {
		return "", ErrNoBody
	}

	// Build email object
	initialMailboxID := opts.MailboxID
	if initialMailboxID == "" {
		initialMailboxID = draftsMailbox.ID
	}

	emailObj := map[string]any{
		"mailboxIds": map[string]bool{initialMailboxID: true},
		"keywords":   map[string]bool{"$draft": true},
		"from":       []map[string]string{{"email": sendFromEmail}},
		"subject":    opts.Subject,
	}

	// Add Sender header when authorization identity differs from From.
	if authIdentityEmail != "" && !strings.EqualFold(authIdentityEmail, sendFromEmail) {
		emailObj["sender"] = []map[string]string{{"email": authIdentityEmail}}
	}

	// Add recipients
	to := make([]map[string]string, len(opts.To))
	for i, addr := range opts.To {
		to[i] = map[string]string{"email": addr}
	}
	emailObj["to"] = to

	if len(opts.CC) > 0 {
		cc := make([]map[string]string, len(opts.CC))
		for i, addr := range opts.CC {
			cc[i] = map[string]string{"email": addr}
		}
		emailObj["cc"] = cc
	}

	if len(opts.BCC) > 0 {
		bcc := make([]map[string]string, len(opts.BCC))
		for i, addr := range opts.BCC {
			bcc[i] = map[string]string{"email": addr}
		}
		emailObj["bcc"] = bcc
	}

	// Add body
	bodyValues := make(map[string]map[string]string)
	if opts.TextBody != "" {
		emailObj["textBody"] = []map[string]string{{"partId": "text", "type": "text/plain"}}
		bodyValues["text"] = map[string]string{"value": opts.TextBody}
	}
	if opts.HTMLBody != "" {
		emailObj["htmlBody"] = []map[string]string{{"partId": "html", "type": "text/html"}}
		bodyValues["html"] = map[string]string{"value": opts.HTMLBody}
	}
	emailObj["bodyValues"] = bodyValues

	// Add attachments if provided
	if len(opts.Attachments) > 0 {
		attachments := make([]map[string]any, len(opts.Attachments))
		for i, att := range opts.Attachments {
			attachments[i] = map[string]any{
				"blobId":      att.BlobID,
				"name":        att.Name,
				"type":        att.Type,
				"disposition": "attachment",
			}
		}
		emailObj["attachments"] = attachments
	}

	// Build submission object
	submissionObj := map[string]any{
		"emailId":    "#draft",
		"identityId": authIdentityID,
	}

	// Only include explicit envelope for non-masked emails.
	// For masked emails, let Fastmail derive the envelope automatically.
	if envelopeFromEmail != "" {
		rcptTo := make([]map[string]string, len(opts.To))
		for i, addr := range opts.To {
			rcptTo[i] = map[string]string{"email": addr}
		}
		submissionObj["envelope"] = map[string]any{
			"mailFrom": map[string]string{"email": envelopeFromEmail},
			"rcptTo":   rcptTo,
		}
	}

	// Include masked email capability if sending from a masked email
	usingCaps := []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail", "urn:ietf:params:jmap:submission"}
	if isMaskedEmail {
		usingCaps = append(usingCaps, maskedEmailNamespace)
	}

	req := &Request{
		Using: usingCaps,
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"create":    map[string]any{"draft": emailObj},
			}, "createEmail"},
			{"EmailSubmission/set", map[string]any{
				"accountId": session.AccountID,
				"create": map[string]any{
					"submission": submissionObj,
				},
				"onSuccessUpdateEmail": map[string]any{
					"#submission": map[string]any{
						"mailboxIds": map[string]bool{sentMailbox.ID: true},
						"keywords":   map[string]bool{"$seen": true},
					},
				},
			}, "submitEmail"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	// Check email creation
	emailResult, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	if notCreated, notCreatedOK := emailResult["notCreated"].(map[string]any); notCreatedOK {
		if errInfo, exists := notCreated["draft"]; exists {
			return "", fmt.Errorf("failed to create email: %v", errInfo)
		}
	}

	// Check email submission
	submissionResult, ok := resp.MethodResponses[1][1].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	if notCreated, ok := submissionResult["notCreated"].(map[string]any); ok {
		if errInfo, exists := notCreated["submission"]; exists {
			return "", fmt.Errorf("failed to submit email: %v", errInfo)
		}
	}

	// Extract submission ID
	if created, ok := submissionResult["created"].(map[string]any); ok {
		if submission, ok := created["submission"].(map[string]any); ok {
			if id, ok := submission["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "unknown", nil
}

// BulkResult contains the result of a bulk operation.
type BulkResult struct {
	Succeeded []string          // IDs that were successfully processed
	Failed    map[string]string // ID -> error message for failures
}

// DeleteEmail moves an email to trash.
func (c *Client) DeleteEmail(ctx context.Context, id string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	// Find trash mailbox
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return err
	}

	var trashMailbox *Mailbox
	for i := range mailboxes {
		if mailboxes[i].Role == "trash" {
			trashMailbox = &mailboxes[i]
			break
		}
	}

	if trashMailbox == nil {
		return ErrNoTrashMailbox
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					id: map[string]any{
						"mailboxIds": map[string]bool{trashMailbox.ID: true},
					},
				},
			}, "moveToTrash"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		if _, exists := notUpdated[id]; exists {
			return fmt.Errorf("failed to delete email")
		}
	}

	return nil
}

// DeleteEmails moves multiple emails to trash in a single JMAP request.
// Returns a BulkResult containing IDs that succeeded and failed.
// Handles partial failures gracefully - some emails may succeed while others fail.
func (c *Client) DeleteEmails(ctx context.Context, ids []string) (*BulkResult, error) {
	// Handle empty/nil input
	if len(ids) == 0 {
		return &BulkResult{
			Succeeded: []string{},
			Failed:    map[string]string{},
		}, nil
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Find trash mailbox
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return nil, err
	}

	var trashMailbox *Mailbox
	for i := range mailboxes {
		if mailboxes[i].Role == "trash" {
			trashMailbox = &mailboxes[i]
			break
		}
	}

	if trashMailbox == nil {
		return nil, ErrNoTrashMailbox
	}

	// Build updates map for all IDs
	updates := make(map[string]any)
	for _, id := range ids {
		updates[id] = map[string]any{
			"mailboxIds": map[string]bool{trashMailbox.ID: true},
		}
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update":    updates,
			}, "moveToTrash"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Parse succeeded and failed IDs
	succeeded, failed := parseBulkUpdateResult(result)

	return &BulkResult{
		Succeeded: succeeded,
		Failed:    failed,
	}, nil
}

// parseBulkUpdateResult extracts succeeded and failed IDs from an Email/set update response.
func parseBulkUpdateResult(result map[string]any) ([]string, map[string]string) {
	succeeded := []string{}
	failed := make(map[string]string)

	// Extract succeeded updates
	if updated, ok := result["updated"].(map[string]any); ok {
		for id := range updated {
			succeeded = append(succeeded, id)
		}
	}

	// Extract failed updates
	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		for id, errInfo := range notUpdated {
			errMsg := "unknown error"
			if errMap, ok := errInfo.(map[string]any); ok {
				errType := getString(errMap, "type")
				errDesc := getString(errMap, "description")
				if errType != "" && errDesc != "" {
					errMsg = errType + ": " + errDesc
				} else if errType != "" {
					errMsg = errType
				} else if errDesc != "" {
					errMsg = errDesc
				}
			}
			failed[id] = errMsg
		}
	}

	return succeeded, failed
}

// MoveEmails moves multiple emails to a target mailbox in a single JMAP request.
// Returns a BulkResult containing IDs that succeeded and failed.
// Handles partial failures gracefully - some emails may succeed while others fail.
func (c *Client) MoveEmails(ctx context.Context, ids []string, targetMailboxID string) (*BulkResult, error) {
	// Handle empty/nil input
	if len(ids) == 0 {
		return &BulkResult{
			Succeeded: []string{},
			Failed:    map[string]string{},
		}, nil
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Build updates map for all IDs
	updates := make(map[string]any)
	for _, id := range ids {
		updates[id] = map[string]any{
			"mailboxIds": map[string]bool{targetMailboxID: true},
		}
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update":    updates,
			}, "moveEmails"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Parse succeeded and failed IDs
	succeeded, failed := parseBulkUpdateResult(result)

	return &BulkResult{
		Succeeded: succeeded,
		Failed:    failed,
	}, nil
}

// MoveEmail moves an email to a target mailbox.
// Note: This is a true MOVE operation - the email will be removed from all
// other mailboxes and placed only in the target mailbox. For emails in
// multiple folders, this may not be desired behavior.
func (c *Client) MoveEmail(ctx context.Context, id, targetMailboxID string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					id: map[string]any{
						"mailboxIds": map[string]bool{targetMailboxID: true},
					},
				},
			}, "moveEmail"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		if _, exists := notUpdated[id]; exists {
			return fmt.Errorf("failed to move email")
		}
	}

	return nil
}

// MarkEmailRead marks an email as read or unread.
// Uses JMAP patch syntax to only modify $seen without affecting other keywords.
func (c *Client) MarkEmailRead(ctx context.Context, id string, read bool) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	// Use JMAP patch syntax: "keywords/$seen" to modify only that flag
	// Setting to true marks as read, setting to null removes the flag (unread)
	var seenValue any
	if read {
		seenValue = true
	} else {
		seenValue = nil // null in JMAP removes the keyword
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					id: map[string]any{
						"keywords/$seen": seenValue,
					},
				},
			}, "updateEmail"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		if _, exists := notUpdated[id]; exists {
			if read {
				return fmt.Errorf("failed to mark email as read")
			}
			return fmt.Errorf("failed to mark email as unread")
		}
	}

	return nil
}

// MarkEmailsRead marks multiple emails as read or unread in a single JMAP request.
func (c *Client) MarkEmailsRead(ctx context.Context, ids []string, read bool) (*BulkResult, error) {
	// Handle empty/nil input
	if len(ids) == 0 {
		return &BulkResult{
			Succeeded: []string{},
			Failed:    map[string]string{},
		}, nil
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// Use JMAP patch syntax: "keywords/$seen" to modify only that flag
	// Setting to true marks as read, setting to null removes the flag (unread)
	var seenValue any
	if read {
		seenValue = true
	} else {
		seenValue = nil // null in JMAP removes the keyword
	}

	// Build updates map for all IDs
	updates := make(map[string]any)
	for _, id := range ids {
		updates[id] = map[string]any{
			"keywords/$seen": seenValue,
		}
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/set", map[string]any{
				"accountId": session.AccountID,
				"update":    updates,
			}, "markRead"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	// Parse succeeded and failed IDs
	succeeded, failed := parseBulkUpdateResult(result)

	return &BulkResult{
		Succeeded: succeeded,
		Failed:    failed,
	}, nil
}

// GetThread retrieves all emails in a thread.
func (c *Client) GetThread(ctx context.Context, threadID string) ([]Email, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	// First, check if threadID is actually an email ID
	actualThreadID := threadID

	// Try to resolve thread ID from email ID
	emailReq := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"ids":        []string{threadID},
				"properties": []string{"threadId"},
			}, "checkEmail"},
		},
	}

	emailResp, err := c.MakeRequest(ctx, emailReq)
	if err == nil {
		if result, ok := emailResp.MethodResponses[0][1].(map[string]any); ok {
			if list, ok := result["list"].([]any); ok && len(list) > 0 {
				if email, ok := list[0].(map[string]any); ok {
					if tid, ok := email["threadId"].(string); ok {
						actualThreadID = tid
					}
				}
			}
		}
	}

	// Get thread with all emails
	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Thread/get", map[string]any{
				"accountId": session.AccountID,
				"ids":       []string{actualThreadID},
			}, "getThread"},
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"#ids":       map[string]any{"resultOf": "getThread", "name": "Thread/get", "path": "/list/*/emailIds"},
				"properties": []string{"id", "subject", "from", "to", "cc", "receivedAt", "preview", "hasAttachment", "keywords", "threadId"},
			}, "emails"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Check if thread was found
	threadResult, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	if notFound, ok := threadResult["notFound"].([]any); ok && len(notFound) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrThreadNotFound, actualThreadID)
	}

	return parseEmailList(resp.MethodResponses[1])
}

// GetEmailAttachments retrieves attachments for an email.
func (c *Client) GetEmailAttachments(ctx context.Context, id string) ([]Attachment, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"ids":        []string{id},
				"properties": []string{"attachments"},
			}, "getAttachments"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	list, ok := result["list"].([]any)
	if !ok || len(list) == 0 {
		return []Attachment{}, nil
	}

	emailData, ok := list[0].(map[string]any)
	if !ok {
		return []Attachment{}, nil
	}

	attachments, ok := emailData["attachments"].([]any)
	if !ok {
		return []Attachment{}, nil
	}

	result_attachments := make([]Attachment, 0, len(attachments))
	for _, item := range attachments {
		att, ok := item.(map[string]any)
		if !ok {
			continue
		}

		attachment := Attachment{
			PartID: getString(att, "partId"),
			BlobID: getString(att, "blobId"),
			Name:   getString(att, "name"),
			Type:   getString(att, "type"),
			Size:   getInt64(att, "size"),
		}
		result_attachments = append(result_attachments, attachment)
	}

	return result_attachments, nil
}

// createIdentity creates a sending identity for the given email.
func (c *Client) createIdentity(ctx context.Context, email string) (string, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return "", err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:submission"},
		MethodCalls: []MethodCall{
			{"Identity/set", map[string]any{
				"accountId": session.AccountID,
				"create": map[string]any{
					"new": map[string]any{
						"email": email,
						"name":  "", // No display name needed
					},
				},
			}, "createIdentity"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	// Check if creation failed
	if notCreated, ok := result["notCreated"].(map[string]any); ok {
		if errInfo, exists := notCreated["new"]; exists {
			return "", fmt.Errorf("failed to create identity: %v", errInfo)
		}
	}

	// Extract created identity ID
	if created, ok := result["created"].(map[string]any); ok {
		if identity, ok := created["new"].(map[string]any); ok {
			return getString(identity, "id"), nil
		}
	}

	return "", fmt.Errorf("identity creation returned unexpected result")
}

// deleteIdentity deletes a sending identity by ID.
func (c *Client) deleteIdentity(ctx context.Context, id string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:submission"},
		MethodCalls: []MethodCall{
			{"Identity/set", map[string]any{
				"accountId": session.AccountID,
				"destroy":   []string{id},
			}, "destroyIdentity"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notDestroyed, ok := result["notDestroyed"].(map[string]any); ok {
		if errInfo, exists := notDestroyed[id]; exists {
			return fmt.Errorf("failed to delete identity: %v", errInfo)
		}
	}

	return nil
}

// GetIdentities retrieves sending identities for the account.
func (c *Client) GetIdentities(ctx context.Context) ([]Identity, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:submission"},
		MethodCalls: []MethodCall{
			{"Identity/get", map[string]any{
				"accountId": session.AccountID,
			}, "identities"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	list, ok := result["list"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected list format")
	}

	identities := make([]Identity, 0, len(list))
	for _, item := range list {
		id, ok := item.(map[string]any)
		if !ok {
			continue
		}

		identity := Identity{
			ID:        getString(id, "id"),
			Name:      getString(id, "name"),
			Email:     getString(id, "email"),
			MayDelete: getBool(id, "mayDelete"),
		}
		identities = append(identities, identity)
	}

	return identities, nil
}

func (c *Client) getDefaultIdentity(ctx context.Context) (*Identity, error) {
	identities, err := c.GetIdentities(ctx)
	if err != nil {
		return nil, err
	}
	if len(identities) == 0 {
		return nil, ErrNoIdentities
	}

	for i := range identities {
		if !identities[i].MayDelete {
			return &identities[i], nil
		}
	}

	return &identities[0], nil
}

// Helper functions for parsing

func parseEmailList(methodResp MethodResponse) ([]Email, error) {
	result, ok := methodResp[1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	list, ok := result["list"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected list format")
	}

	emails := make([]Email, 0, len(list))
	for _, item := range list {
		emailData, ok := item.(map[string]any)
		if !ok {
			continue
		}
		emails = append(emails, *parseEmail(emailData))
	}

	return emails, nil
}

func parseEmail(data map[string]any) *Email {
	email := &Email{
		ID:            getString(data, "id"),
		ThreadID:      getString(data, "threadId"),
		Subject:       getString(data, "subject"),
		ReceivedAt:    getString(data, "receivedAt"),
		Preview:       getString(data, "preview"),
		HasAttachment: getBool(data, "hasAttachment"),
		// Initialize all slice fields to empty (not nil) so JSON
		// serialization produces [] instead of null.
		From:        []EmailAddress{},
		To:          []EmailAddress{},
		CC:          []EmailAddress{},
		BCC:         []EmailAddress{},
		ReplyTo:     []EmailAddress{},
		TextBody:    []BodyPart{},
		HTMLBody:    []BodyPart{},
		Attachments: []Attachment{},
		MessageID:   []string{},
		InReplyTo:   []string{},
		References:  []string{},
	}

	// Parse addresses
	if from, ok := data["from"].([]any); ok {
		email.From = parseAddresses(from)
	}
	if to, ok := data["to"].([]any); ok {
		email.To = parseAddresses(to)
	}
	if cc, ok := data["cc"].([]any); ok {
		email.CC = parseAddresses(cc)
	}
	if bcc, ok := data["bcc"].([]any); ok {
		email.BCC = parseAddresses(bcc)
	}
	if replyTo, ok := data["replyTo"].([]any); ok {
		email.ReplyTo = parseAddresses(replyTo)
	}

	// Parse threading headers
	if messageId, ok := data["messageId"].([]any); ok {
		email.MessageID = parseStringArray(messageId)
	}
	if inReplyTo, ok := data["inReplyTo"].([]any); ok {
		email.InReplyTo = parseStringArray(inReplyTo)
	}
	if references, ok := data["references"].([]any); ok {
		email.References = parseStringArray(references)
	}

	// Parse keywords
	if keywords, ok := data["keywords"].(map[string]any); ok {
		email.Keywords = make(map[string]bool)
		for k, v := range keywords {
			if b, ok := v.(bool); ok {
				email.Keywords[k] = b
			}
		}
	}

	// Parse mailboxIds
	if mailboxIds, ok := data["mailboxIds"].(map[string]any); ok {
		email.MailboxIDs = make(map[string]bool)
		for k, v := range mailboxIds {
			if b, ok := v.(bool); ok {
				email.MailboxIDs[k] = b
			}
		}
	}

	// Parse bodyValues
	if bodyValues, ok := data["bodyValues"].(map[string]any); ok {
		email.BodyValues = make(map[string]BodyValue)
		for k, v := range bodyValues {
			if bv, ok := v.(map[string]any); ok {
				email.BodyValues[k] = BodyValue{
					Value: getString(bv, "value"),
				}
			}
		}
	}

	// Parse body parts
	if textBody, ok := data["textBody"].([]any); ok {
		email.TextBody = parseBodyParts(textBody)
	}
	if htmlBody, ok := data["htmlBody"].([]any); ok {
		email.HTMLBody = parseBodyParts(htmlBody)
	}

	// Parse attachments
	if attachments, ok := data["attachments"].([]any); ok {
		email.Attachments = make([]Attachment, 0, len(attachments))
		for _, item := range attachments {
			if att, ok := item.(map[string]any); ok {
				email.Attachments = append(email.Attachments, Attachment{
					PartID: getString(att, "partId"),
					BlobID: getString(att, "blobId"),
					Name:   getString(att, "name"),
					Type:   getString(att, "type"),
					Size:   getInt64(att, "size"),
				})
			}
		}
	}

	return email
}

func parseAddresses(addrs []any) []EmailAddress {
	result := make([]EmailAddress, 0, len(addrs))
	for _, item := range addrs {
		if addr, ok := item.(map[string]any); ok {
			result = append(result, EmailAddress{
				Name:  getString(addr, "name"),
				Email: getString(addr, "email"),
			})
		}
	}
	return result
}

func parseBodyParts(parts []any) []BodyPart {
	result := make([]BodyPart, 0, len(parts))
	for _, item := range parts {
		if part, ok := item.(map[string]any); ok {
			result = append(result, BodyPart{
				PartID: getString(part, "partId"),
				Type:   getString(part, "type"),
			})
		}
	}
	return result
}

func parseStringArray(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// SearchSnippet contains highlighted search result context.
type SearchSnippet struct {
	EmailID string `json:"emailId"`
	Subject string `json:"subject,omitempty"` // Highlighted subject if matched
	Preview string `json:"preview,omitempty"` // Highlighted preview/body snippet
}

// SearchEmailsWithSnippets searches for emails and returns highlighted snippets.
func (c *Client) SearchEmailsWithSnippets(ctx context.Context, searchFilter *EmailSearchFilter, limit int) ([]Email, []SearchSnippet, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, nil, err
	}

	filter := map[string]any{}
	if searchFilter != nil {
		filter = searchFilter.ToJMAPFilter()
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/query", map[string]any{
				"accountId": session.AccountID,
				"filter":    filter,
				"sort":      []map[string]any{{"property": "receivedAt", "isAscending": false}},
				"limit":     limit,
			}, "query"},
			{"Email/get", map[string]any{
				"accountId":  session.AccountID,
				"#ids":       map[string]any{"resultOf": "query", "name": "Email/query", "path": "/ids"},
				"properties": []string{"id", "subject", "from", "to", "cc", "receivedAt", "preview", "hasAttachment", "keywords", "threadId"},
			}, "emails"},
			{"SearchSnippet/get", map[string]any{
				"accountId": session.AccountID,
				"filter":    filter,
				"#emailIds": map[string]any{"resultOf": "query", "name": "Email/query", "path": "/ids"},
			}, "snippets"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	emails, err := parseEmailList(resp.MethodResponses[1])
	if err != nil {
		return nil, nil, err
	}

	snippets, err := parseSearchSnippets(resp.MethodResponses[2])
	if err != nil {
		return nil, nil, err
	}

	return emails, snippets, nil
}

func parseSearchSnippets(methodResp MethodResponse) ([]SearchSnippet, error) {
	result, ok := methodResp[1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid SearchSnippet/get response: expected map, got %T", methodResp[1])
	}

	list, ok := result["list"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid SearchSnippet/get response: expected list array, got %T", result["list"])
	}

	snippets := make([]SearchSnippet, 0, len(list))
	for _, item := range list {
		s, ok := item.(map[string]any)
		if !ok {
			continue
		}
		snippets = append(snippets, SearchSnippet{
			EmailID: getString(s, "emailId"),
			Subject: getString(s, "subject"),
			Preview: getString(s, "preview"),
		})
	}

	return snippets, nil
}

// CreateMailboxOpts contains options for creating a mailbox.
type CreateMailboxOpts struct {
	Name     string // Required: name of the mailbox
	ParentID string // Optional: parent mailbox ID (empty for root)
}

// CreateMailbox creates a new mailbox (folder).
func (c *Client) CreateMailbox(ctx context.Context, opts CreateMailboxOpts) (*Mailbox, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("mailbox name is required")
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	mailboxObj := map[string]any{
		"name": opts.Name,
	}

	if opts.ParentID != "" {
		mailboxObj["parentId"] = opts.ParentID
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Mailbox/set", map[string]any{
				"accountId": session.AccountID,
				"create":    map[string]any{"new": mailboxObj},
			}, "createMailbox"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	if notCreated, ok := result["notCreated"].(map[string]any); ok {
		if errInfo, exists := notCreated["new"]; exists {
			return nil, fmt.Errorf("failed to create mailbox: %v", errInfo)
		}
	}

	if created, ok := result["created"].(map[string]any); ok {
		if mb, ok := created["new"].(map[string]any); ok {
			return &Mailbox{
				ID:   getString(mb, "id"),
				Name: opts.Name,
			}, nil
		}
	}

	return nil, fmt.Errorf("mailbox created but ID not returned")
}

// DeleteMailbox deletes a mailbox by ID.
// If the mailbox contains emails, they will be moved to trash.
func (c *Client) DeleteMailbox(ctx context.Context, id string) error {
	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Mailbox/set", map[string]any{
				"accountId":             session.AccountID,
				"destroy":               []string{id},
				"onDestroyRemoveEmails": false, // Move emails to trash instead of deleting
			}, "deleteMailbox"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notDestroyed, ok := result["notDestroyed"].(map[string]any); ok {
		if errInfo, exists := notDestroyed[id]; exists {
			return fmt.Errorf("failed to delete mailbox: %v", errInfo)
		}
	}

	return nil
}

// RenameMailbox renames a mailbox.
func (c *Client) RenameMailbox(ctx context.Context, id, newName string) error {
	if newName == "" {
		return fmt.Errorf("new name is required")
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Mailbox/set", map[string]any{
				"accountId": session.AccountID,
				"update": map[string]any{
					id: map[string]any{
						"name": newName,
					},
				},
			}, "renameMailbox"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	if notUpdated, ok := result["notUpdated"].(map[string]any); ok {
		if errInfo, exists := notUpdated[id]; exists {
			return fmt.Errorf("failed to rename mailbox: %v", errInfo)
		}
	}

	return nil
}

// ForwardEmailOpts contains options for forwarding an email.
type ForwardEmailOpts struct {
	To   []string // Required: recipient addresses
	From string   // Optional: override sender (default: auto-detect masked email)
	Body string   // Optional: message to prepend to forwarded content
}

// ForwardFromSource indicates how the From address was chosen for a forward.
type ForwardFromSource string

const (
	// ForwardFromExplicit indicates the user explicitly specified the From address via --from flag.
	ForwardFromExplicit ForwardFromSource = "explicit"
	// ForwardFromMasked indicates the From address was automatically detected from a masked email
	// found in the original email's recipients.
	ForwardFromMasked ForwardFromSource = "masked"
	// ForwardFromDefault indicates the default identity was used because no masked email was
	// detected and no explicit --from flag was provided.
	ForwardFromDefault ForwardFromSource = "default"
)

// ResolveForwardFrom determines the From address used when forwarding.
func (c *Client) ResolveForwardFrom(ctx context.Context, original *Email, opts ForwardEmailOpts) (string, ForwardFromSource, error) {
	if opts.From != "" {
		return opts.From, ForwardFromExplicit, nil
	}

	maskedFrom := c.findMaskedEmailRecipient(ctx, original)
	if maskedFrom != "" {
		return maskedFrom, ForwardFromMasked, nil
	}

	identities, err := c.GetIdentities(ctx)
	if err != nil {
		return "", "", err
	}
	if len(identities) == 0 {
		return "", "", ErrNoIdentities
	}

	// Match SendEmail behavior: use primary (non-deletable) identity if available.
	defaultIdentity := identities[0]
	for _, id := range identities {
		if !id.MayDelete {
			defaultIdentity = id
			break
		}
	}

	return defaultIdentity.Email, ForwardFromDefault, nil
}

// ForwardEmail forwards an email to new recipients.
// Automatically uses the masked email address if the original was received on one.
// Includes all original attachments.
func (c *Client) ForwardEmail(ctx context.Context, original *Email, opts ForwardEmailOpts) (string, error) {
	if len(opts.To) == 0 {
		return "", fmt.Errorf("at least one recipient is required")
	}

	// Determine the From address
	// If not specified, check if original was received on a masked email
	fromAddress := opts.From
	if fromAddress == "" {
		maskedFrom := c.findMaskedEmailRecipient(ctx, original)
		if maskedFrom != "" {
			fromAddress = maskedFrom
		}
	}

	// Build forward subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "fwd:") {
		subject = "Fwd: " + subject
	}

	// Build forward body with header
	textBody, htmlBody := buildForwardBody(original, opts.Body)

	// Prepare attachments (reuse existing blob IDs)
	var attachments []AttachmentOpts
	for _, att := range original.Attachments {
		attachments = append(attachments, AttachmentOpts{
			BlobID: att.BlobID,
			Name:   att.Name,
			Type:   att.Type,
		})
	}

	// Send the forwarded email
	sendOpts := SendEmailOpts{
		To:          opts.To,
		Subject:     subject,
		TextBody:    textBody,
		HTMLBody:    htmlBody,
		From:        fromAddress,
		Attachments: attachments,
	}

	return c.SendEmail(ctx, sendOpts)
}

// buildForwardBody creates the forward message body with headers.
func buildForwardBody(original *Email, prependBody string) (textBody, htmlBody string) {
	// Format the date in human-readable RFC1123Z format
	receivedTime, err := time.Parse(time.RFC3339, original.ReceivedAt)
	var dateStr string
	if err == nil {
		dateStr = receivedTime.Format(time.RFC1123Z) // "Mon, 02 Jan 2006 15:04:05 -0700"
	} else {
		dateStr = original.ReceivedAt // fallback to raw value
	}

	// Build forward header
	forwardHeader := fmt.Sprintf(
		"---------- Forwarded message ---------\n"+
			"From: %s\n"+
			"Date: %s\n"+
			"Subject: %s\n"+
			"To: %s\n",
		formatAddressList(original.From),
		dateStr,
		original.Subject,
		formatAddressList(original.To),
	)

	if len(original.CC) > 0 {
		forwardHeader += fmt.Sprintf("Cc: %s\n", formatAddressList(original.CC))
	}
	forwardHeader += "\n"

	// Get original body content
	var originalTextBody string
	if len(original.TextBody) > 0 && len(original.BodyValues) > 0 {
		for _, part := range original.TextBody {
			if body, ok := original.BodyValues[part.PartID]; ok {
				originalTextBody = body.Value
				break
			}
		}
	}

	var originalHTMLBody string
	if len(original.HTMLBody) > 0 && len(original.BodyValues) > 0 {
		for _, part := range original.HTMLBody {
			if body, ok := original.BodyValues[part.PartID]; ok {
				originalHTMLBody = body.Value
				break
			}
		}
	}

	// Build text body
	if prependBody != "" {
		textBody = prependBody + "\n\n" + forwardHeader + originalTextBody
	} else {
		textBody = forwardHeader + originalTextBody
	}

	// Build HTML body if original had HTML
	if originalHTMLBody != "" {
		htmlForwardHeader := strings.ReplaceAll(forwardHeader, "\n", "<br>\n")
		if prependBody != "" {
			escapedBody := html.EscapeString(prependBody)
			htmlBody = "<p>" + strings.ReplaceAll(escapedBody, "\n", "<br>") + "</p><br>\n" +
				"<div style=\"border-left: 2px solid #ccc; padding-left: 10px; margin-left: 5px;\">\n" +
				"<p style=\"color: #666;\">" + htmlForwardHeader + "</p>\n" +
				originalHTMLBody + "\n</div>"
		} else {
			htmlBody = "<div style=\"border-left: 2px solid #ccc; padding-left: 10px; margin-left: 5px;\">\n" +
				"<p style=\"color: #666;\">" + htmlForwardHeader + "</p>\n" +
				originalHTMLBody + "\n</div>"
		}
	}

	return textBody, htmlBody
}

// formatAddressList formats a list of email addresses for display.
func formatAddressList(addrs []EmailAddress) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, len(addrs))
	for i, addr := range addrs {
		if addr.Name != "" {
			parts[i] = fmt.Sprintf("%s <%s>", addr.Name, addr.Email)
		} else {
			parts[i] = addr.Email
		}
	}
	return strings.Join(parts, ", ")
}

// ImportEmailOpts contains options for importing an email.
type ImportEmailOpts struct {
	BlobID     string          // Required: blob ID of uploaded .eml file
	MailboxIDs map[string]bool // Required: mailboxes to add email to
	Keywords   map[string]bool // Optional: keywords like $seen, $flagged
	ReceivedAt string          // Optional: override received date (RFC3339)
}

// GetThreadMessageCounts returns the message count for each thread ID.
// This batches the request to avoid N+1 queries.
func (c *Client) GetThreadMessageCounts(ctx context.Context, threadIDs []string) (map[string]int, error) {
	if len(threadIDs) == 0 {
		return map[string]int{}, nil
	}

	// Deduplicate thread IDs
	seen := make(map[string]bool)
	uniqueIDs := make([]string, 0, len(threadIDs))
	for _, id := range threadIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	if len(uniqueIDs) == 0 {
		return map[string]int{}, nil
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Thread/get", map[string]any{
				"accountId": session.AccountID,
				"ids":       uniqueIDs,
			}, "threads"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	counts := make(map[string]int)
	if list, ok := result["list"].([]any); ok {
		for _, item := range list {
			if thread, ok := item.(map[string]any); ok {
				threadID := getString(thread, "id")
				if emailIDs, ok := thread["emailIds"].([]any); ok {
					counts[threadID] = len(emailIDs)
				}
			}
		}
	}

	return counts, nil
}

// ImportEmail imports a raw RFC 5322 email message into mailboxes.
// First upload the .eml file using UploadBlob, then call this with the blob ID.
func (c *Client) ImportEmail(ctx context.Context, opts ImportEmailOpts) (string, error) {
	if opts.BlobID == "" {
		return "", fmt.Errorf("blobId is required")
	}
	if len(opts.MailboxIDs) == 0 {
		return "", fmt.Errorf("at least one mailbox is required")
	}

	session, err := c.GetSession(ctx)
	if err != nil {
		return "", err
	}

	emailObj := map[string]any{
		"blobId":     opts.BlobID,
		"mailboxIds": opts.MailboxIDs,
	}

	if len(opts.Keywords) > 0 {
		emailObj["keywords"] = opts.Keywords
	}

	if opts.ReceivedAt != "" {
		emailObj["receivedAt"] = opts.ReceivedAt
	}

	req := &Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: []MethodCall{
			{"Email/import", map[string]any{
				"accountId": session.AccountID,
				"emails":    map[string]any{"import1": emailObj},
			}, "importEmail"},
		},
	}

	resp, err := c.MakeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	result, ok := resp.MethodResponses[0][1].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	if notCreated, ok := result["notCreated"].(map[string]any); ok {
		if errInfo, exists := notCreated["import1"]; exists {
			return "", fmt.Errorf("failed to import email: %v", errInfo)
		}
	}

	if created, ok := result["created"].(map[string]any); ok {
		if email, ok := created["import1"].(map[string]any); ok {
			if id, ok := email["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("email imported but ID not returned")
}
