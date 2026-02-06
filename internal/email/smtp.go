package email

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

type SMTPSender struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func NewSMTPSender(host string, port int, user, password, from string) *SMTPSender {
	return &SMTPSender{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
	}
}

func (s *SMTPSender) SendDownloadLink(email, downloadURL string, stats string) {
	// Run in background to not block worker
	go func() {
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

		// Setup Authentication
		// Note: Simply using PlainAuth. For modern services use appropriate auth (e.g., App Passwords for Gmail).
		var auth smtp.Auth
		if s.User != "" && s.Password != "" {
			auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
		}

		subject := "Your Database Export is Ready"
		body := fmt.Sprintf("Hello,\n\nYour export job has completed successfully.\n\nStats: %s\n\nDownload Link:\n%s\n\nThis link will expire depending on your storage policy.\n", stats, downloadURL)

		msg := []byte(fmt.Sprintf("To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s\r\n", email, subject, body))

		slog.Info("Sending email via SMTP", "to", email, "host", s.Host)

		// SMTP Send
		// In production, consider using a library like 'gomail' for better multipart/HTML handling,
		// but 'net/smtp' is sufficient for simple text emails.
		err := smtp.SendMail(addr, auth, s.From, []string{email}, msg)
		if err != nil {
			// Check if it's a "Sender address rejected" or similar.
			// Often local dev servers (like MailHog) don't need auth.
			slog.Error("Failed to send email", "error", err, "to", email)
		} else {
			slog.Info("Email sent successfully", "to", email)
		}
	}()
}

func (s *SMTPSender) SendWithAttachment(emailAddr, filename string, content []byte, stats string) {
	go func() {
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

		var auth smtp.Auth
		if s.User != "" && s.Password != "" {
			auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
		}

		boundary := "MyBoundarySeparator"
		subject := "Your Database Export is Ready (Attached)"
		bodyText := fmt.Sprintf("Hello,\n\nYour export job has completed successfully.\n\nStats: %s\n\nPlease find the export attached.\n", stats)

		// Headers
		headers := make(map[string]string)
		headers["To"] = emailAddr
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "multipart/mixed; boundary=\"" + boundary + "\""

		headerStr := ""
		for k, v := range headers {
			headerStr += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		headerStr += "\r\n"

		// Body Part
		msg := headerStr
		msg += fmt.Sprintf("--%s\r\n", boundary)
		msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
		msg += "\r\n" + bodyText + "\r\n"

		// Attachment Part
		contentType := "application/octet-stream"
		if strings.HasSuffix(filename, ".csv") {
			contentType = "text/csv"
		} else if strings.HasSuffix(filename, ".gz") {
			contentType = "application/gzip"
		}

		encoded := base64.StdEncoding.EncodeToString(content)

		msg += fmt.Sprintf("--%s\r\n", boundary)
		msg += fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", contentType, filename)
		msg += "Content-Transfer-Encoding: base64\r\n"
		msg += fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename)
		msg += "\r\n"

		// Split Base64 lines (RFC 2045 limit 76 chars)
		// Simple approach: write logic to chunk it
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			msg += encoded[i:end] + "\r\n"
		}

		msg += fmt.Sprintf("\r\n--%s--", boundary)

		slog.Info("Sending email with attachment via SMTP", "to", emailAddr, "size", len(content))

		err := smtp.SendMail(addr, auth, s.From, []string{emailAddr}, []byte(msg))
		if err != nil {
			slog.Error("Failed to send email", "error", err, "to", emailAddr)
		} else {
			slog.Info("Email sent successfully", "to", emailAddr)
		}
	}()
}
