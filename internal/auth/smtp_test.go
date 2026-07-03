package auth

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wneessen/go-mail"
)

func TestNewSMTPSender_Validation(t *testing.T) {
	cases := []struct {
		name    string
		cfg     SMTPConfig
		wantErr bool
	}{
		{"missing host", SMTPConfig{From: "a@b.com", Port: 587}, true},
		{"missing from", SMTPConfig{Host: "smtp.example.com", Port: 587}, true},
		{"valid no auth", SMTPConfig{Host: "smtp.example.com", From: "a@b.com", Port: 587}, false},
		{"valid with auth", SMTPConfig{Host: "smtp.example.com", From: "a@b.com", Port: 465, Username: "u", Password: "p"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewSMTPSender(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.client == nil {
				t.Error("client not initialized")
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	s, err := NewSMTPSender(SMTPConfig{Host: "smtp.example.com", From: "no-reply@sshbin.test", Port: 587})
	if err != nil {
		t.Fatalf("NewSMTPSender: %v", err)
	}

	msg, err := s.buildMessage("user@example.com", "123456")
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}

	if from := msg.GetFromString(); len(from) != 1 || !strings.Contains(from[0], "no-reply@sshbin.test") {
		t.Errorf("From = %v, want no-reply@sshbin.test", from)
	}
	if to := msg.GetToString(); len(to) != 1 || !strings.Contains(to[0], "user@example.com") {
		t.Errorf("To = %v, want user@example.com", to)
	}
	if subj := msg.GetGenHeader(mail.HeaderSubject); len(subj) != 1 || subj[0] != "Your sshbin sign-in code" {
		t.Errorf("Subject = %v", subj)
	}

	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if !strings.Contains(buf.String(), "123456") {
		t.Error("rendered message does not contain the code")
	}
}

func TestBuildMessage_InvalidAddress(t *testing.T) {
	s, err := NewSMTPSender(SMTPConfig{Host: "smtp.example.com", From: "no-reply@sshbin.test", Port: 587})
	if err != nil {
		t.Fatalf("NewSMTPSender: %v", err)
	}
	if _, err := s.buildMessage("not-an-email", "123456"); err == nil {
		t.Error("want error for invalid destination address")
	}
}
