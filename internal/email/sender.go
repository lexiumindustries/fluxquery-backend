package email

import (
	"log/slog"
	"time"
)

type Sender interface {
	SendDownloadLink(email, downloadURL string, stats string)
	SendWithAttachment(email, filename string, content []byte, stats string)
}

type LogSender struct{}

func NewLogSender() *LogSender {
	return &LogSender{}
}

// SendDownloadLink sends an email asynchronously.
// In a real implementation, this would use an SMTP server or SES.
// Here we log it and simulate non-blocking behavior.
func (s *LogSender) SendDownloadLink(email, downloadURL string, stats string) {
	go func() {
		// Simulate network latency and retry logic
		time.Sleep(100 * time.Millisecond)
		slog.Info("EMAIL SENT",
			"to", email,
			"url", downloadURL,
			"stats", stats,
		)
	}()
}

func (s *LogSender) SendWithAttachment(email, filename string, content []byte, stats string) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		slog.Info("EMAIL SENT WITH ATTACHMENT",
			"to", email,
			"filename", filename,
			"size", len(content),
			"stats", stats,
		)
	}()
}
