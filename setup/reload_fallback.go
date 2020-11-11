// +build !darwin,!linux,!freebsd

package main

import (
	"errors"
)

func Reload() error {
	return errors.New("the reload command is only supported on POSIX systems")
}
