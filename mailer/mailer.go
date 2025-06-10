package mailer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

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
	To        string
	Subject   string
	View      string
	Variables any
}

func (s *Sender) render(view string, vars any) (string, string, error) {
	htmlPath := filepath.Join("views", "mails", "html", view+".html")
	txtPath := filepath.Join("views", "mails", view+".txt")

	htmlTpl, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", "", fmt.Errorf("read html template: %w", err)
	}
	txtTpl, err := os.ReadFile(txtPath)
	if err != nil {
		return "", "", fmt.Errorf("read text template: %w", err)
	}

	htmlT, err := template.New("html").Parse(string(htmlTpl))
	if err != nil {
		return "", "", fmt.Errorf("parse html template: %w", err)
	}
	txtT, err := template.New("txt").Parse(string(txtTpl))
	if err != nil {
		return "", "", fmt.Errorf("parse text template: %w", err)
	}

	var htmlBuf, txtBuf bytes.Buffer
	if err := htmlT.Execute(&htmlBuf, vars); err != nil {
		return "", "", err
	}
	if err := txtT.Execute(&txtBuf, vars); err != nil {
		return "", "", err
	}
	return htmlBuf.String(), txtBuf.String(), nil
}

// Send sends the message using the configured SMTP server.
func (s *Sender) Send(ctx context.Context, m Message) error {
	htmlBody, textBody, err := s.render(m.View, m.Variables)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	client, err := mail.NewClient(addr, mail.WithSMTPAuth(s.cfg.Username, s.cfg.Password))
	if err != nil {
		return fmt.Errorf("create mail client: %w", err)
	}
	msg := mail.NewMsg()
	if err := msg.From(s.cfg.From); err != nil {
		return err
	}
	if err := msg.To(m.To); err != nil {
		return err
	}
	msg.Subject(m.Subject)
	msg.SetBodyString(mail.TypeTextPlain, textBody)
	msg.AddAlternative(mail.TypeTextHTML, htmlBody)
	return client.DialAndSend(ctx, msg)
}
