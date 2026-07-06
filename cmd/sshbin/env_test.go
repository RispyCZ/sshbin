package main

import (
	"testing"
	"time"
)

// stubEnv replaces the env lookup with a fixed map for the duration of a test.
func stubEnv(t *testing.T, m map[string]string) {
	t.Helper()
	prev := lookupEnv
	lookupEnv = func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
	t.Cleanup(func() { lookupEnv = prev })
}

// captureFatal replaces fatalf with a recorder so tests can assert on
// misconfiguration without exiting the process.
func captureFatal(t *testing.T) *bool {
	t.Helper()
	called := false
	prev := fatalf
	fatalf = func(string, ...any) { called = true }
	t.Cleanup(func() { fatalf = prev })
	return &called
}

func TestEnvName(t *testing.T) {
	cases := map[string]string{
		"sftp-listen": "SSHBIN_SFTP_LISTEN",
		"base-url":    "SSHBIN_BASE_URL",
		"db":          "SSHBIN_DB",
		"dev":         "SSHBIN_DEV",
	}
	for in, want := range cases {
		if got := envName(in); got != want {
			t.Errorf("envName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnvDefaultString(t *testing.T) {
	t.Run("env set overrides default", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_BASE_URL": "https://prod.example"})
		if got := envDefaultString("base-url", "http://localhost"); got != "https://prod.example" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("env unset falls back to default", func(t *testing.T) {
		stubEnv(t, map[string]string{})
		if got := envDefaultString("base-url", "http://localhost"); got != "http://localhost" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("empty env value wins over default", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_BASE_URL": ""})
		if got := envDefaultString("base-url", "http://localhost"); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestEnvDefaultInt(t *testing.T) {
	t.Run("valid env parsed", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_SMTP_PORT": " 465 "})
		if got := envDefaultInt("smtp-port", 587); got != 465 {
			t.Errorf("got %d, want 465", got)
		}
	})
	t.Run("unset falls back", func(t *testing.T) {
		stubEnv(t, map[string]string{})
		if got := envDefaultInt("smtp-port", 587); got != 587 {
			t.Errorf("got %d, want 587", got)
		}
	})
	t.Run("malformed is fatal", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_SMTP_PORT": "notanumber"})
		fatal := captureFatal(t)
		envDefaultInt("smtp-port", 587)
		if !*fatal {
			t.Error("expected fatalf on malformed int")
		}
	})
}

func TestSMTPPassword(t *testing.T) {
	t.Run("prefixed var used", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_SMTP_PASSWORD": "secret"})
		if got := smtpPassword(); got != "secret" {
			t.Errorf("got %q, want secret", got)
		}
	})
	t.Run("unset is empty", func(t *testing.T) {
		stubEnv(t, map[string]string{})
		if got := smtpPassword(); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestEnvDefaultBool(t *testing.T) {
	t.Run("valid env parsed", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_DEV": "true"})
		if got := envDefaultBool("dev", false); !got {
			t.Error("got false, want true")
		}
	})
	t.Run("unset falls back", func(t *testing.T) {
		stubEnv(t, map[string]string{})
		if got := envDefaultBool("dev", true); !got {
			t.Error("got false, want true")
		}
	})
	t.Run("malformed is fatal", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_DEV": "maybe"})
		fatal := captureFatal(t)
		envDefaultBool("dev", false)
		if !*fatal {
			t.Error("expected fatalf on malformed bool")
		}
	})
}

func TestEnvDefaultDuration(t *testing.T) {
	t.Run("valid env parsed", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_PRUNE_INTERVAL": " 30m "})
		if got := envDefaultDuration("prune-interval", time.Hour); got != 30*time.Minute {
			t.Errorf("got %v, want 30m", got)
		}
	})
	t.Run("unset falls back", func(t *testing.T) {
		stubEnv(t, map[string]string{})
		if got := envDefaultDuration("prune-interval", time.Hour); got != time.Hour {
			t.Errorf("got %v, want 1h", got)
		}
	})
	t.Run("malformed is fatal", func(t *testing.T) {
		stubEnv(t, map[string]string{"SSHBIN_PRUNE_INTERVAL": "notaduration"})
		fatal := captureFatal(t)
		envDefaultDuration("prune-interval", time.Hour)
		if !*fatal {
			t.Error("expected fatalf on malformed duration")
		}
	})
}
