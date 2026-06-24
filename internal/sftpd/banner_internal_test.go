package sftpd

import (
	"strings"
	"testing"
)

func TestSetupBanner(t *testing.T) {
	out := setupBanner("myfile.log", "http://localhost:8080/setup/abc")

	if !strings.Contains(out, "myfile.log") {
		t.Error("banner missing file name")
	}
	if !strings.Contains(out, "http://localhost:8080/setup/abc") {
		t.Error("banner missing URL")
	}
	if !strings.Contains(out, "\x1b[") {
		t.Error("banner missing ANSI color codes")
	}
}

func TestSanitizeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"clean.log", "clean.log"},
		{"\x1b[31mevil\x1b[0m.log", "evil.log"},
		{"break\r\nout.log", "breakout.log"},
		{"tab\there.log", "tabhere.log"},
	}
	for _, c := range cases {
		if got := sanitizeName(c.in); got != c.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
