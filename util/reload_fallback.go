// +build !darwin,!linux,!freebsd

package util

import (
	"os"
)

func Reload(logger *Logger) {
	logger.PrintError("Error: The reload command is only supported on POSIX systems\n")
	os.Exit(1)
}
