package strcase

import "testing"

// TestScreamingSnake pins the conversion against what iancoleman/strcase
// produced, which this package replaced. The name a secret is stored under
// decides which secret a lookup finds, so the answers can't drift.
func TestScreamingSnake(t *testing.T) {
	for name, want := range map[string]string{
		"DB_DSN":              "DB_DSN",
		"db_dsn":              "DB_DSN",
		"dbDsn":               "DB_DSN",
		"db-dsn":              "DB_DSN",
		"db.dsn":              "DB_DSN",
		"db dsn":              "DB_DSN",
		"  db dsn  ":          "DB_DSN",
		"fooBar":              "FOO_BAR",
		"JSONData":            "JSON_DATA",
		"HTTPServer":          "HTTP_SERVER",
		"userID":              "USER_ID",
		"v2Beta":              "V_2_BETA",
		"x1y2z3":              "X_1_Y_2_Z_3",
		"1foo":                "1_FOO",
		"9lives":              "9_LIVES",
		"foo!bar":             "FOO!BAR",
		"a__b":                "A__B",
		"--a--":               "__A__",
		"":                    "",
		"a":                   "A",
		"A":                   "A",
		"_":                   "_",
		"MiXeD_case-Thing.42": "MI_XE_D_CASE_THING_42",
	} {
		if got := ScreamingSnake(name); got != want {
			t.Errorf("ScreamingSnake(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestScreamingSnakeBytes covers input which isn't ASCII. It's copied
// through a byte at a time rather than decoded, so only the ASCII letters
// are raised. Storing such a name is a bad idea anyway: the env command
// refuses to print one.
func TestScreamingSnakeBytes(t *testing.T) {
	for name, want := range map[string]string{
		"ünïcödé":  "üNïCöDé",
		"naïve_id": "NAïVE_ID",
	} {
		if got := ScreamingSnake(name); got != want {
			t.Errorf("ScreamingSnake(%q) = %q, want %q", name, got, want)
		}
	}
}
