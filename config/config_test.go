package config_test

import (
	"testing"

	"github.com/pganalyze/collector/config"
)

type testItem struct {
	input    string
	expected string
}

var tests = []testItem{
	{"postgres://user:password@example.com", "postgres://user@example.com"},
	{"postgres://user:password@example.com?sslmode=verify-full", "postgres://user@example.com?sslmode=verify-full"},
	{"postgres://user@example.com", "postgres://user@example.com"},
	{string([]byte{0x7f}), "<unparsable>"},
	{"postgres://user:pass:word@example.com", "postgres://user@example.com"},
	{"", ""},
}

func TestGetDbURLRedacted(t *testing.T) {
	var config config.ServerConfig

	for _, item := range tests {
		config.DbURL = item.input
		if redacted := config.GetDbURLRedacted(); redacted != item.expected {
			t.Errorf("want %s; got %s", item.expected, redacted)
		}
	}
}
