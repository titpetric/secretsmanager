package main

import (
	"os/exec"
	"testing"
)

// TestShellQuote covers quoting a value for a .env file.
func TestShellQuote(t *testing.T) {
	for value, want := range map[string]string{
		"plain":            `"plain"`,
		`with"quote`:       `"with\"quote"`,
		"with$var":         `"with\$var"`,
		"with`backtick":    "\"with\\`backtick\"",
		`with\backslash`:   `"with\\backslash"`,
		`c:\dir\"$x`:       `"c:\\dir\\\"\$x"`,
		"multi\nline":      "\"multi\nline\"",
		"with'singlequote": `"with'singlequote"`,
	} {
		if got := shellQuote(value); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", value, got, want)
		}
	}
}

// TestShellQuoteRoundTrip feeds the quoted value back through a shell, which
// is what `secretsmanager env >> .env` ends up doing.
func TestShellQuoteRoundTrip(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	values := []string{
		"plain",
		"user:password@hostname",
		`with"quote`,
		"with$var",
		"with`backtick`",
		`with\backslash`,
		`all\of"them$and` + "`this`",
		"multi\nline",
		"trailing\\",
		"$(echo pwned)",
	}

	for _, value := range values {
		script := "SECRET=" + shellQuote(value) + "; printf %s \"$SECRET\""

		out, err := exec.Command(sh, "-c", script).Output()
		if err != nil {
			t.Errorf("sh -c %q: %v", script, err)
			continue
		}
		if string(out) != value {
			t.Errorf("round trip of %q gave %q (script: %s)", value, out, script)
		}
	}
}
