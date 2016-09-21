// +build !darwin,!linux,!freebsd

package util

import (
	"fmt"
	"os"
)

func Reload() {
	fmt.Printf("Error: The reload command is only supported on POSIX systems\n")
	os.Exit(1)
}
