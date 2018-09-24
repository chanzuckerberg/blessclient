package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// TokenProvider lets us simulate some interactivity during
// ssh flow
// HACK HACK: This probably deserves a longer explanation.
func TokenProvider(prompt string) func() (string, error) {
	// While running through the ssh config (match exec, ProxyCommand, etc)
	// ssh commandeers stdin and stdout for internal use.
	// this makes it hard to do things like asking for an MFA token.
	// Fortunately for us, stderr(2) still points to our tty
	// and is used to communicate with the user.
	// We exploit this quirk to provide a simple interactive mode
	// that allows us to request MFA tokens from users
	return func() (string, error) {
		newStdIn := os.NewFile(uintptr(syscall.Stderr), "/dev/stdin")
		newStdOut := os.NewFile(uintptr(syscall.Stderr), "/dev/stdout")
		defer newStdIn.Close()
		defer newStdOut.Close()

		fmt.Fprintf(newStdOut, "AWS MFA token:")
		reader := bufio.NewReader(newStdIn)
		text, err := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		return text, err
	}
}
