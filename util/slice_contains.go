package util

// helper function to check if a string is contained in a slice
// to avoid starting multiple servers for the same endpoint
func SliceContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
