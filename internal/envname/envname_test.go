package envname

import "testing"

// TestName covers the environment name a secret is used under.
func TestName(t *testing.T) {
	for name, want := range map[string]string{
		"DB_DSN":  "DB_DSN",
		"db_dsn":  "DB_DSN",
		"dbDsn":   "DB_DSN",
		"db-dsn":  "DB_DSN",
		"foo bar": "FOO_BAR",
	} {
		if got := Name(name); got != want {
			t.Errorf("Name(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestValid covers the names the create command accepts.
func TestValid(t *testing.T) {
	for name, want := range map[string]bool{
		"DB_DSN":  true,
		"_DB_DSN": true,
		"DB_DSN2": true,
		"":        false,
		"1_FOO":   false,
		"FOO!BAR": false,
		"FOO BAR": false,
		"FOO-BAR": false,
	} {
		if got := Valid(name); got != want {
			t.Errorf("Valid(%q) = %v, want %v", name, got, want)
		}
	}
}
