package config_test

import (
	"testing"

	"github.com/pganalyze/collector/config"
)

func TestGetDbURLRedacted(t *testing.T) {
	type testItem struct {
		input    string
		expected string
	}

	tests := []testItem{
		{"postgres://user:password@example.com", "postgres://user@example.com"},
		{"postgres://user:password@example.com?sslmode=verify-full", "postgres://user@example.com?sslmode=verify-full"},
		{"postgres://user@example.com", "postgres://user@example.com"},
		{string([]byte{0x7f}), "<unparsable>"},
		{"postgres://user:pass:word@example.com", "postgres://user@example.com"},
		{"", ""},
	}
	var config config.ServerConfig

	for _, item := range tests {
		config.DbURL = item.input
		if redacted := config.GetDbURLRedacted(); redacted != item.expected {
			t.Errorf("want %s; got %s", item.expected, redacted)
		}
	}
}

func TestGetEffectiveDbUsername(t *testing.T) {
	type testItem struct {
		systemType string
		dbURL      string
		expected   string
	}
	tests := []testItem{
		{"amazon_rds", "postgres://user:password@example.com", "user"},
		{"planetscale", "postgres://user.abc1234:password@example.com", "user"},
		{"planetscale", "postgres://user.abc1234%7Creplica:password@example.com", "user"},
		{"planetscale", "postgres://foo.bar.abc1234%7Creplica:password@example.com", "foo.bar"},
		{"planetscale", "postgres://foo.bar.abc1234%7Creplica.2:password@example.com", "foo.bar"},
	}

	var config config.ServerConfig

	for _, item := range tests {
		config.SystemType = item.systemType
		config.DbURL = item.dbURL
		if username := config.GetEffectiveDbUsername(); username != item.expected {
			t.Errorf("want %s; got %s", item.expected, username)
		}
	}
}
