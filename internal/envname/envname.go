// Package envname names a secret after the environment variable it's used
// under, which is how one secret is told from another.
package envname

import "github.com/titpetric/secretsmanager/internal/strcase"

// Name returns the environment variable name a secret is used under. Names
// which differ only in case or in their separators share one, which is what
// makes them one secret.
func Name(name string) string {
	return strcase.ScreamingSnake(name)
}

// Valid reports whether name is usable as an environment variable name.
// Anything else can be stored, but not printed as an environment variable.
func Valid(name string) bool {
	if name == "" {
		return false
	}

	for i, r := range name {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}

	return true
}
