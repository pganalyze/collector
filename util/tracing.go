package util

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// TimeFromStr returns a time of the given value in string.
// The value is the value of time either as a floating point number of seconds since the Epoch, or it can also be a integer number of seconds.
func TimeFromStr(value string) (time.Time, error) {
	IntegerAndDecimal := strings.SplitN(value, ".", 2)
	secInStr := IntegerAndDecimal[0]
	sec, err := strconv.ParseInt(secInStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if len(IntegerAndDecimal) == 1 {
		return time.Unix(sec, 0).UTC(), nil
	}
	decimalInStr := IntegerAndDecimal[1]
	if decimalInStr == "" {
		decimalInStr = "0"
	}
	if len(decimalInStr) > 9 {
		// decimal length shouldn't be more than nanoseconds (9)
		return time.Time{}, errors.New("decimal length is longer than nanoseconds (9)")
	}
	nsecInStr := rightPad(decimalInStr, 9, "0")
	nsec, err := strconv.ParseInt(nsecInStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, nsec).UTC(), nil
}

// rightPad returns the string that is right padded with the given pad string.
func rightPad(str string, length int, pad string) string {
	if len(str) >= length {
		return str
	}
	padding := strings.Repeat(pad, length-len(str))
	return str + padding
}
