//go:build !darwin && !linux && !freebsd
// +build !darwin,!linux,!freebsd

package util

import "errors"

func Reload(logger *Logger) (reloadedPid int, err error) {
	return -1, errors.New("the reload command is only supported on POSIX systems")
}

func ReloadSelf() error {
	return errors.New("automatic reloads are only supported on POSIX systems")
}
