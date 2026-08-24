package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// readLine prompts until a line which isn't blank is read, and returns it
// with only the line ending removed: a secret may be padded with spaces on
// purpose, and trimming it would store a value nobody asked for.
//
// The end of the input is an error rather than a reason to prompt again.
// Prompts go to stderr, so the output of a command stays usable when it's
// redirected.
func (s *SecretsManager) readLine(prompt string) (string, error) {
	for {
		fmt.Fprint(os.Stderr, prompt+" ")

		line, err := s.in.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) != "" {
			return line, nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", errors.New("unexpected end of input")
			}
			return "", err
		}
	}
}

// readName prompts for a name, which carries no significant whitespace.
func (s *SecretsManager) readName(prompt string) (string, error) {
	line, err := s.readLine(prompt)
	return strings.TrimSpace(line), err
}
