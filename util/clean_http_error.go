package util

import (
	"errors"
	"fmt"
	"regexp"
)

var regex = regexp.MustCompile("(?i): (get|post|patch) ")

// Removes duplicate URLs added by retryablehttp
func CleanHTTPError(error error) error {
	message := fmt.Sprintf("%v", error)
	array := regex.Split(message, -1)
	return errors.New(array[len(array)-1])
}
