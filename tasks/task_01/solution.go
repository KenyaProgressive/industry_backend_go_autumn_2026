package main

import (
	"fmt"
	"strings"
)

func greet(name string) string {
	nameFormatted := strings.TrimSpace(name)
	if nameFormatted == " " || nameFormatted == "" {
		return "Hello, World!"
	}
	return fmt.Sprintf("Hello, %s!", nameFormatted)
}
