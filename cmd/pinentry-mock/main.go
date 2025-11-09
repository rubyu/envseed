package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// pinentry-mock is a minimal Assuan-speaking pinentry replacement for tests.
// Behavior is controlled via environment variables:
//
//	PINENTRY_MOCK_PASSPHRASE: the passphrase to return on GETPIN (default: "testpass")
//	PINENTRY_MOCK_ACTION: "" (success), "cancel" (operation cancelled), "error" (generic error)
//
// It acknowledges all unknown commands with OK to keep gpg-agent happy.
func main() {
	pass := os.Getenv("PINENTRY_MOCK_PASSPHRASE")
	if pass == "" {
		pass = "testpass"
	}
	action := strings.ToLower(os.Getenv("PINENTRY_MOCK_ACTION"))

	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	// Assuan servers greet first
	fmt.Fprintln(out, "OK Pleased to meet you")
	out.Flush()

	for in.Scan() {
		line := in.Text()
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "GETPIN"):
			switch action {
			case "cancel":
				fmt.Fprintln(out, "ERR 83886179 Operation cancelled")
			case "error":
				// GPG_ERR_INV_DATA
				fmt.Fprintln(out, "ERR 67109133 Invalid data")
			default:
				fmt.Fprintf(out, "D %s\n", pass)
				fmt.Fprintln(out, "OK")
			}
		case strings.HasPrefix(upper, "CONFIRM"):
			if action == "cancel" {
				fmt.Fprintln(out, "ERR 83886179 Operation cancelled")
			} else {
				fmt.Fprintln(out, "OK")
			}
		case strings.HasPrefix(upper, "BYE"):
			fmt.Fprintln(out, "OK")
			out.Flush()
			return
		default:
			// Acknowledge other setup commands: SETDESC, SETPROMPT, OPTION, RESET, GETINFO, etc.
			fmt.Fprintln(out, "OK")
		}

		out.Flush()
	}
}
