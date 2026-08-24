// Package strcase converts a name to the SCREAMING_SNAKE_CASE an
// environment variable is written in.
//
// It's a port of ToScreamingSnake from github.com/iancoleman/strcase, MIT
// licensed, Copyright (c) 2015 Ian Coleman, cut down to the one conversion
// this repository makes. The behaviour is kept exactly, because the name a
// secret is stored under decides which secret a lookup finds.
package strcase

import "strings"

// ScreamingSnake converts a name to SCREAMING_SNAKE_CASE. A change of case,
// and each of space, underscore, hyphen and dot, separate two words.
func ScreamingSnake(s string) string {
	s = strings.TrimSpace(s)

	out := strings.Builder{}
	out.Grow(len(s) + 2)

	// The conversion reads bytes, not runes: anything outside ASCII is
	// copied through as the bytes it's made of.
	for i, v := range []byte(s) {
		isUpper := v >= 'A' && v <= 'Z'
		isLower := v >= 'a' && v <= 'z'
		if isLower {
			v -= 'a' - 'A'
		}

		// Look at the next byte, so a word ending can be told from an
		// acronym running into one: JSONData is JSON and Data.
		if i+1 < len(s) {
			next := s[i+1]
			isNumber := v >= '0' && v <= '9'
			nextIsUpper := next >= 'A' && next <= 'Z'
			nextIsLower := next >= 'a' && next <= 'z'
			nextIsNumber := next >= '0' && next <= '9'

			if (isUpper && (nextIsLower || nextIsNumber)) ||
				(isLower && (nextIsUpper || nextIsNumber)) ||
				(isNumber && (nextIsUpper || nextIsLower)) {
				if isUpper && nextIsLower {
					if previousIsUpper := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; previousIsUpper {
						out.WriteByte('_')
					}
				}

				out.WriteByte(v)
				if isLower || isNumber || nextIsNumber {
					out.WriteByte('_')
				}
				continue
			}
		}

		if v == ' ' || v == '_' || v == '-' || v == '.' {
			out.WriteByte('_')
		} else {
			out.WriteByte(v)
		}
	}

	return out.String()
}
