package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func readline(prompt string) (value string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt + " ")
		value, _ = reader.ReadString('\n')
		value = strings.TrimSpace(value)
		if value != "" {
			break
		}
	}
	return value
}
