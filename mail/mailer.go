package mail

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	txtTemplate "text/template"
	"time"

	mail "github.com/wneessen/go-mail"

	"github.com/alexisvisco/goframe/core/configuration"
)

// Sender is responsible for sending emails using SMTP.
type Sender struct {
	cfg configuration.Mail
}

// NewSender creates a new Sender.
func NewSender(cfg configuration.Mail) *Sender {
	return &Sender{cfg: cfg}
}

// Message represents an email message.
type Message struct {
	To        []string
	Bcc       []string
	Cc        []string
	From      string // Optional, if not set, will use cfg.DefaultMailFrom
	ReplyTo   string // Optional, if not set, will use cfg.ReplyTo
	Subject   string
	View      string
	Variables any

	Attachments []Attachment // Optional attachments
}

type Attachment struct {
	Filename string
	Data     string
	MimeType string // MIME type of the attachment
}

func NewAttachment(filename string, data []byte, mimeType string) Attachment {
	return Attachment{
		Filename: filename,
		Data:     base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
	}
}

func (s *Sender) render(view string) (*template.Template, *txtTemplate.Template, error) {
	htmlPath := filepath.Join("views", "mails", "html", view+".html")
	txtPath := filepath.Join("views", "mails", view+".txt.tmpl")

	htmlTpl, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read html template: %w", err)
	}
	txtTpl, err := os.ReadFile(txtPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read text template: %w", err)
	}

	htmlT, err := template.New("html").Parse(string(htmlTpl))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html template: %w", err)
	}
	txtT, err := txtTemplate.New("txt").Parse(string(txtTpl))
	if err != nil {
		return nil, nil, fmt.Errorf("parse text template: %w", err)
	}

	return htmlT, txtT, nil
}

// getSMTPAuthType returns the appropriate SMTPAuthType based on the configuration.
func (s *Sender) getSMTPAuthType() mail.SMTPAuthType {
	switch s.cfg.AuthType {
	case configuration.MailAuthTypePlain:
		return mail.SMTPAuthPlain
	case configuration.MailAuthTypeLogin:
		return mail.SMTPAuthLogin
	case configuration.MailAuthTypeCRAMMD5:
		return mail.SMTPAuthCramMD5
	default:
		return ""
	}
}

// getTLSPolicy returns the appropriate TLSPolicy based on the configuration.
func (s *Sender) getTLSPolicy() mail.TLSPolicy {
	switch s.cfg.TLSPolicy {
	case configuration.TLSPolicyNone:
		return mail.NoTLS
	case configuration.TLSPolicyMandatory:
		return mail.TLSMandatory
	default:
		return mail.TLSOpportunistic
	}
}

// Send sends the message using the configured SMTP server.
func (s *Sender) Send(ctx context.Context, m Message) error {
	htmlTemplate, textTemplate, err := s.render(m.View)
	if err != nil {
		return err
	}

	clientOptions := []mail.Option{
		mail.WithPort(s.cfg.Port),
		mail.WithTimeout(10 * time.Second),
	}

	// Set TLS policy based on configuration
	tlsPolicy := s.getTLSPolicy()

	// Override TLS policy for development/testing environments (like Mailpit)
	if (s.cfg.Host == "localhost" || s.cfg.Host == "127.0.0.1" || s.cfg.Port == 1025) && s.cfg.TLSPolicy == "" {
		// Only override if not explicitly set
		tlsPolicy = mail.NoTLS
	}

	clientOptions = append(clientOptions, mail.WithTLSPolicy(tlsPolicy))

	// Only add authentication if username and password are provided
	// and auth type is not none
	if s.cfg.Username != "" && s.cfg.Password != "" && s.cfg.AuthType != configuration.MailAuthTypeNone {
		authType := s.getSMTPAuthType()
		clientOptions = append(clientOptions,
			mail.WithSMTPAuth(authType),
			mail.WithUsername(s.cfg.Username),
			mail.WithPassword(s.cfg.Password),
		)
	}

	client, err := mail.NewClient(s.cfg.Host, clientOptions...)
	if err != nil {
		return fmt.Errorf("create mail client: %w", err)
	}

	from := s.cfg.DefaultFrom
	if m.From != "" {
		from = m.From
	}

	msg := mail.NewMsg()
	if err := msg.From(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if err := msg.To(m.To...); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	if err := msg.Bcc(m.Bcc...); err != nil {
		return fmt.Errorf("invalid bcc address: %w", err)
	}
	if err := msg.Cc(m.Cc...); err != nil {
		return fmt.Errorf("invalid cc address: %w", err)
	}

	if m.ReplyTo != "" {
		if err := msg.ReplyTo(m.ReplyTo); err != nil {
			return fmt.Errorf("invalid reply-to address: %w", err)
		}
	}

	if len(m.Attachments) > 0 {
		for _, attachment := range m.Attachments {
			var data []byte
			var err error

			// Otherwise, use the raw bytes directly
			if len(attachment.Data) > 0 {
				// Try to decode as base64 first
				data, err = base64.StdEncoding.DecodeString(attachment.Data)
				if err != nil {
					// If base64 decoding fails, assume it's raw binary data
					data = []byte(attachment.Data)
				}
			} else {
				return fmt.Errorf("attachment %s has no data", attachment.Filename)
			}

			msg.AttachReader(attachment.Filename, bytes.NewReader(data), func(file *mail.File) {
				contentType := "application/octet-stream"
				if attachment.MimeType != "" {
					contentType = attachment.MimeType
				}

				file.Header.Set("Content-Type", contentType)
			})
		}
	}

	msg.Subject(m.Subject)

	errTxt := msg.SetBodyTextTemplate(textTemplate, m.Variables)
	errHtml := msg.AddAlternativeHTMLTemplate(htmlTemplate, m.Variables)
	if err := cmp.Or(errTxt, errHtml); err != nil {
		return fmt.Errorf("unable to set body: %w", err)
	}

	return client.DialAndSendWithContext(ctx, msg)
}
