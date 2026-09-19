// Command hash-password prints the bcrypt hash of a password, in the same
// format the backend expects in the `user.password_hash` column. Use it to
// generate ADMIN_PASSWORD_HASH before a real deployment, or to rotate any
// account's password afterwards with a manual UPDATE.
//
// Usage:
//
//	go run ./cmd/hash-password "MyNewPassword2026*"
//	go run ./cmd/hash-password            # prompts for the password instead
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := passwordFromArgsOrPrompt()
	if password == "" {
		fmt.Fprintln(os.Stderr, "a non-empty password is required")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hashing the password failed:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}

func passwordFromArgsOrPrompt() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	fmt.Fprint(os.Stderr, "Password: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}
