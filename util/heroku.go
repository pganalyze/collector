package util

import "os"

func IsHeroku() bool {
	return os.Getenv("DYNO") != "" && os.Getenv("PORT") != ""
}
