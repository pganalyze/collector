// +build !darwin,!linux,!freebsd

package util

import (
	"errors"
)

func ReloadCollector() error {
	return errors.New("the reload command is only supported on POSIX systems")
}
