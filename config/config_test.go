package config_test

import (
	"testing"
	"time"

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

func TestLogDownloadInterval(t *testing.T) {
	type testItem struct {
		interval             int
		expectedDuration     time.Duration
		expectedWindow       time.Duration
		expectedMaxParseSize int
	}

	const mb = 1024 * 1024
	tests := []testItem{
		// Unset (e.g. a programmatically constructed config) behaves like the default
		{0, 30 * time.Second, 2 * time.Minute, 10 * mb},
		{30, 30 * time.Second, 2 * time.Minute, 10 * mb},
		{60, 1 * time.Minute, 3 * time.Minute, 20 * mb},
		{120, 2 * time.Minute, 5 * time.Minute, 40 * mb},
		// The parsing size counts whole 30 second units, so a partial unit rounds down
		// and 100s gets the same budget as 90s would
		{100, 100 * time.Second, 4*time.Minute + 20*time.Second, 30 * mb},
		// Parsing size scaling reaches the cap at 150s (5 units)
		{150, 150 * time.Second, 6 * time.Minute, 50 * mb},
		{600, 10 * time.Minute, 21 * time.Minute, 50 * mb},
		// Out of range values are rejected when reading the config, but clamped here
		{900, 10 * time.Minute, 21 * time.Minute, 50 * mb},
		{-1, 30 * time.Second, 2 * time.Minute, 10 * mb},
	}

	for _, item := range tests {
		var config config.ServerConfig
		config.LogDownloadInterval = item.interval

		if duration := config.LogDownloadIntervalDuration(); duration != item.expectedDuration {
			t.Errorf("%d: interval: want %s; got %s", item.interval, item.expectedDuration, duration)
		}
		if window := config.LogDownloadWindow(); window != item.expectedWindow {
			t.Errorf("%d: window: want %s; got %s", item.interval, item.expectedWindow, window)
		}
		if size := config.MaxLogParsingSize(); size != item.expectedMaxParseSize {
			t.Errorf("%d: max parsing size: want %d; got %d", item.interval, item.expectedMaxParseSize, size)
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
