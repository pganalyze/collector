package util

import "time"

func StringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func BoolPtrToBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func TimePtrToUnixTimestamp(ptr *time.Time) int64 {
	if ptr == nil {
		return 0
	}
	return ptr.Unix()
}

func IntPtrToString(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
