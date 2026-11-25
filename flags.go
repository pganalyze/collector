package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Each of these flags selects a separate code path; only allow one
var modeFlags = []string{
	"test",
	"test-logs",
	"test-explain",
	"dry-run",
	"dry-run-logs",
	"generate-stats-helper-sql",
	"generate-explain-analyze-helper-sql",
	"analyze-logfile",
	"filter-logfile",
	"debug-logs",
	"discover-log-location",
}

const daemonMode = ""

var testRunModes = []string{"test", "test-logs", "test-explain", "dry-run", "dry-run-logs"}
var logfileModes = []string{"analyze-logfile", "filter-logfile"}
var collectionModes = []string{daemonMode, "test", "dry-run"}

// Flags that only have an effect in some modes and the modes they apply to
var modifierFlags = []struct {
	flag  string
	modes []string
}{
	{"reload", slices.Concat([]string{"reload"}, testRunModes)},
	// --test is the only mode that reloads by default
	{"no-reload", []string{"test"}},
	{"trace", testRunModes},
	{"test-section", testRunModes},
	{"benchmark", []string{daemonMode}},
	{"write-heap-profile", []string{daemonMode, "debug-logs"}},
	// Only these modes collect full snapshots without updating the state file
	{"force-state-update", []string{"test", "dry-run"}},
	{"filter-log-secret", []string{"filter-logfile"}},
	{"analyze-debug-classifications", []string{"analyze-logfile"}},
	{"logfile-prefix", logfileModes},
	{"logfile-tz", logfileModes},
	{"generate-explain-analyze-helper-role", []string{"generate-explain-analyze-helper-sql"}},
	{"no-postgres-relations", collectionModes},
	{"no-postgres-settings", collectionModes},
	{"no-postgres-locks", collectionModes},
	{"no-system-information", collectionModes},
	{"no-logs", collectionModes},
	{"pidfile", []string{daemonMode, "debug-logs"}},
}

// Pairs of flags where one would silently override the other.
var conflictingFlags = [][2]string{
	// Syslog output has its own format
	{"syslog", "json-logs"},
	// Inherently opposites
	{"reload", "no-reload"},
	{"quiet", "verbose"},
	{"quiet", "very-verbose"},
}

// checkFlagCombination returns an error on invalid combinations
func checkFlagCombination(givenFlags map[string]bool) error {
	var modes []string
	for _, f := range modeFlags {
		if givenFlags[f] {
			modes = append(modes, f)
		}
	}
	if len(modes) > 1 {
		return fmt.Errorf("%s can't be combined", joinList(flagNames(modes), "and"))
	}
	for _, pair := range conflictingFlags {
		if givenFlags[pair[0]] && givenFlags[pair[1]] {
			return fmt.Errorf("%s can't be combined", joinList(flagNames(pair[:]), "and"))
		}
	}
	mode := daemonMode
	switch {
	case len(modes) == 1:
		mode = modes[0]
	case givenFlags["reload"]:
		mode = "reload"
	}
	for _, m := range modifierFlags {
		if !givenFlags[m.flag] || slices.Contains(m.modes, mode) {
			continue
		}
		applies := fmt.Sprintf("--%s only applies to %s", m.flag, describeModes(m.modes, m.flag))
		if mode == daemonMode {
			return errors.New(applies)
		}
		return fmt.Errorf("--%s can't be combined with --%s (%s)", m.flag, mode, applies)
	}
	return nil
}

// describeModes lists the modes a modifier flag applies to, omitting the
// implicit mode set by the flag itself (if applicable).
func describeModes(modes []string, flag string) string {
	var names []string
	for _, m := range modes {
		switch m {
		case flag:
		case daemonMode:
			names = append(names, "the collector daemon")
		default:
			names = append(names, "--"+m)
		}
	}
	return joinList(names, "or")
}

func flagNames(names []string) []string {
	flags := make([]string, len(names))
	for i, name := range names {
		flags[i] = "--" + name
	}
	return flags
}

// joinList formats items as "a and b" or "a, b or c".
func joinList(items []string, conjunction string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + conjunction + " " + items[len(items)-1]
}
