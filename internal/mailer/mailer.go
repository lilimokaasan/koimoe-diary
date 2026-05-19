package mailer

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"sakurairo-go/internal/config"
	"sakurairo-go/internal/models"
)

type Sender interface {
	Send(message Message) error
	SendNewComment(comment models.Comment, post models.Post, site config.Site, postURL string) error
}

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type SMTPMailer struct {
	cfg config.Mail
}

func NewSMTP(cfg config.Mail) *SMTPMailer {
	return &SMTPMailer{cfg: normalize(cfg)}
}

func (m *SMTPMailer) Enabled() bool {
	return m.cfg.Enabled && m.cfg.Host != "" && m.cfg.Port > 0 && m.cfg.From != "" && m.cfg.AdminEmail != ""
}

func (m *SMTPMailer) Send(message Message) error {
	if !m.Enabled() {
		return errors.New("mail is not configured")
	}
	if strings.TrimSpace(message.To) == "" {
		return errors.New("recipient is required")
	}
	payload, err := m.build(message)
	if err != nil {
		return err
	}
	return m.deliver(message.To, payload)
}

func (m *SMTPMailer) SendNewComment(comment models.Comment, post models.Post, site config.Site, postURL string) error {
	if !m.Enabled() {
		return errors.New("mail is not configured")
	}
	subject := "[" + site.Name + "] New comment on " + post.Title
	content := html.EscapeString(comment.Content)
	content = strings.ReplaceAll(content, "\n", "<br>")
	author := html.EscapeString(comment.Author)
	postTitle := html.EscapeString(post.Title)
	body := fmt.Sprintf(`
<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;line-height:1.7;color:#4b4350">
  <h2 style="color:#e674a0;margin:0 0 12px">A new comment arrived</h2>
  <p><strong>%s</strong> left a note on <strong>%s</strong>.</p>
  <div style="border-left:4px solid #fb98c0;background:#fff6fa;padding:14px 16px;border-radius:8px">%s</div>
  <p style="font-size:13px;color:#8f8791">Open the post: <a href="%s">%s</a></p>
</div>`, author, postTitle, content, html.EscapeString(postURL), html.EscapeString(postURL))
	text := fmt.Sprintf("New comment by %s on %s:\n\n%s\n\n%s", comment.Author, post.Title, comment.Content, postURL)
	return m.Send(Message{
		To:      m.cfg.AdminEmail,
		Subject: subject,
		Text:    text,
		HTML:    body,
	})
}

func (m *SMTPMailer) build(message Message) ([]byte, error) {
	from := mail.Address{Name: m.cfg.FromName, Address: m.cfg.From}
	to := mail.Address{Address: message.To}
	boundary := "sakurairo-" + fmt.Sprint(time.Now().UnixNano())
	var buf bytes.Buffer
	writeHeader(&buf, "From", from.String())
	writeHeader(&buf, "To", to.String())
	writeHeader(&buf, "Subject", mime.QEncoding.Encode("utf-8", safeHeader(message.Subject)))
	writeHeader(&buf, "MIME-Version", "1.0")
	writeHeader(&buf, "Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	buf.WriteString("\r\n")
	writePart(&buf, boundary, "text/plain; charset=utf-8", message.Text)
	writePart(&buf, boundary, "text/html; charset=utf-8", message.HTML)
	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes(), nil
}

func (m *SMTPMailer) deliver(to string, payload []byte) error {
	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprint(m.cfg.Port))
	var client *smtp.Client
	var err error
	if m.cfg.TLSMode == "implicit" {
		conn, tlsErr := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12})
		if tlsErr != nil {
			return tlsErr
		}
		client, err = smtp.NewClient(conn, m.cfg.Host)
	} else {
		client, err = smtp.Dial(addr)
	}
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Quit(); err != nil {
			log.Printf("mail quit: %v", err)
		}
	}()
	if m.cfg.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}
	if m.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func writeHeader(buf *bytes.Buffer, key string, value string) {
	buf.WriteString(key + ": " + value + "\r\n")
}

func writePart(buf *bytes.Buffer, boundary string, contentType string, body string) {
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: " + contentType + "\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	buf.WriteString(body + "\r\n")
}

func safeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func normalize(cfg config.Mail) config.Mail {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.AdminEmail = strings.TrimSpace(cfg.AdminEmail)
	cfg.TLSMode = strings.ToLower(strings.TrimSpace(cfg.TLSMode))
	if cfg.TLSMode == "" {
		cfg.TLSMode = "starttls"
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return cfg
}
