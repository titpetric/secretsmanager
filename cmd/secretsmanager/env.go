package main

import "strings"

// shellQuoteReplacer escapes the characters a shell still acts on inside
// double quotes. Backslash has to be replaced along with the rest, in the
// single pass a Replacer makes, or it would escape its own replacements.
var shellQuoteReplacer = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	`$`, `\$`,
	"`", "\\`",
)

// shellQuote renders a value as a double quoted shell string, which is the
// form a .env file takes.
func shellQuote(value string) string {
	return `"` + shellQuoteReplacer.Replace(value) + `"`
}
