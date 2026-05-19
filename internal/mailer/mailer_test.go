package mailer

import (
	"strings"
	"testing"

	"sakurairo-go/internal/config"
)

func TestBuildMessageUsesUTF8AlternativeParts(t *testing.T) {
	m := NewSMTP(config.Mail{
		Enabled:    true,
		Host:       "smtp.example.com",
		Port:       587,
		From:       "noreply@example.com",
		FromName:   "KoiMoe Diary",
		AdminEmail: "admin@example.com",
	})
	payload, err := m.build(Message{
		To:      "admin@example.com",
		Subject: "新评论",
		Text:    "hello",
		HTML:    "<p>hello</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(payload)
	for _, want := range []string{
		"Content-Type: multipart/alternative;",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Type: text/html; charset=utf-8",
		"Subject: =?utf-8?",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("payload missing %q:\n%s", want, got)
		}
	}
}

func TestSafeHeaderStripsLineBreaks(t *testing.T) {
	got := safeHeader("Hello\r\nBcc: bad@example.com")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("header still contains line break: %q", got)
	}
}
