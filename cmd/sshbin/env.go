package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// envName maps a flag name to its environment variable counterpart, e.g.
// "sftp-listen" becomes "SSHBIN_SFTP_LISTEN".
func envName(flagName string) string {
	return "SSHBIN_" + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// lookupEnv and fatalf are package-level indirections so tests can stub the
// environment and intercept fatal misconfiguration instead of exiting.
var (
	lookupEnv = os.LookupEnv
	fatalf    = func(msg string, kv ...any) { log.Fatal(msg, kv...) }
)

// envDefaultString returns the value of the flag's env var if set, otherwise def.
func envDefaultString(flagName, def string) string {
	if v, ok := lookupEnv(envName(flagName)); ok {
		return v
	}
	return def
}

// envDefaultInt behaves like envDefaultString but parses the env value as an int.
// A malformed value is fatal so misconfiguration fails fast at startup.
func envDefaultInt(flagName string, def int) int {
	v, ok := lookupEnv(envName(flagName))
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		fatalf("invalid integer env var", "var", envName(flagName), "value", v, "err", err)
		return def
	}
	return n
}

// envDefaultBool behaves like envDefaultString but parses the env value as a bool
// (1/t/true/0/f/false, case-insensitive). A malformed value is fatal.
func envDefaultBool(flagName string, def bool) bool {
	v, ok := lookupEnv(envName(flagName))
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		fatalf("invalid boolean env var", "var", envName(flagName), "value", v, "err", err)
		return def
	}
	return b
}

// envDefaultDuration behaves like envDefaultString but parses the env value as a
// Go duration (e.g. "1h", "72h"). A malformed value is fatal.
func envDefaultDuration(flagName string, def time.Duration) time.Duration {
	v, ok := lookupEnv(envName(flagName))
	if !ok {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		fatalf("invalid duration env var", "var", envName(flagName), "value", v, "err", err)
		return def
	}
	return d
}

// smtpPassword resolves the SMTP password from SSHBIN_SMTP_PASSWORD. It is a
// secret with no flag counterpart.
func smtpPassword() string {
	v, _ := lookupEnv("SSHBIN_SMTP_PASSWORD")
	return v
}

// flagString registers a string flag whose default falls back to SSHBIN_<NAME>.
// Precedence when resolved: explicit flag > env var > def.
func flagString(name, def, usage string) *string {
	return flag.String(name, envDefaultString(name, def), usage)
}

// flagInt registers an int flag whose default falls back to SSHBIN_<NAME>.
func flagInt(name string, def int, usage string) *int {
	return flag.Int(name, envDefaultInt(name, def), usage)
}

// flagBool registers a bool flag whose default falls back to SSHBIN_<NAME>.
func flagBool(name string, def bool, usage string) *bool {
	return flag.Bool(name, envDefaultBool(name, def), usage)
}

// flagDuration registers a duration flag whose default falls back to SSHBIN_<NAME>.
func flagDuration(name string, def time.Duration, usage string) *time.Duration {
	return flag.Duration(name, envDefaultDuration(name, def), usage)
}
