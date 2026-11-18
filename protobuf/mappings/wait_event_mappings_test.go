// Tests for the wait event mapping files in this directory, which the
// pganalyze app uses to encode wait events reported by pg_stat_activity.
//
// The app stores only the wait event enum value and infers the wait event
// type back from it via wait_event_to_type.json (the reverse
// mapping). That only works if every wait event name the collector can send
// belongs to exactly one wait event type. The one name where that is not
// true is "SyncRep": since PG 13 Postgres reports it both as LWLock
// (contention on the SyncRep LWLock) and as IPC (waiting for the synchronous
// standby to confirm). The collector keeps sending the wire wait_event_type
// for backward compatibility, and the app resolves the enum value from the
// (name, type) pair using wait_event_type_resolver.json, so LWLock SyncRep
// maps to WAIT_EVENT_LWLOCK_SYNC_REP_LOCK (132) and IPC SyncRep to
// WAIT_EVENT_SYNC_REP (734). After ingest the type column can be dropped,
// because each of those values now implies its own type.
//
// These tests enforce that invariant instead of trusting it:
//
//   - TestWaitEventTypeOfEventJSON regenerates (with -update) and verifies
//     the reverse mapping from the "// WaitEventType: X" sections in the
//     proto. The canonical type is the section the enum value lives in, so
//     events that Postgres moved between types keep their original label
//     (e.g. buffer_io stays LWLock, ProcSignalBarrier stays IO).
//   - TestWaitEventMappingFiles checks internal consistency of the mapping
//     files, and that the only names shared by enum values of different
//     types are the ones in wait_event_type_resolver.json.
//   - TestPostgresWaitEventNamesAreUnique fetches upstream wait_event_names.txt
//     for the supported Postgres branches and fails if a name appears under
//     two wait event types without a resolver entry. It is network-bound and
//     therefore skipped unless PGANALYZE_CHECK_POSTGRES_WAIT_EVENT_NAMES=1
//     (the CI target check_wait_event_mappings sets it).
package mappings

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	protoPath = "../compact_activity_snapshot.proto"

	reverseMappingPath        = "wait_event_to_type.json"
	waitEventNamesMappingPath = "wait_event.json"
	waitEventTypeMappingPath  = "wait_event_type.json"
	waitEventTypeResolverPath = "wait_event_type_resolver.json"

	// wait_event_names.txt only exists from PG 17 onward; older branches
	// keep the wait event names in C sources, so there is nothing to fetch
	// for them here
	upstreamNamesFile = "src/backend/utils/activity/wait_event_names.txt"
)

// update writes the generated reverse mapping instead of only verifying it,
// via: go test ./protobuf/mappings -run TestWaitEventTypeOfEventJSON -update
var update = flag.Bool("update", false, "regenerate wait_event_to_type.json from the proto")

// Postgres branches whose wait_event_names.txt must be fetchable (all
// branches since PG 17, when the file was introduced).
var requiredPostgresBranches = []string{
	"master",
	"REL_19_STABLE",
	"REL_18_STABLE",
	"REL_17_STABLE",
}

// waitEventTypeByValue is the reverse mapping: wait event enum value ->
// wait event type name, parsed from the "// WaitEventType: X" section
// markers in compact_activity_snapshot.proto. Values without a marker (only
// WAIT_EVENT_UNKNOWN = 0) are absent.
func waitEventTypeByValue(t *testing.T) map[int]string {
	t.Helper()

	protoBytes, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("failed to read %s: %s", protoPath, err)
	}

	var (
		inWaitEventEnum bool
		currentType     string
		byValue         = map[int]string{}
	)

	sectionMarker := regexp.MustCompile(`^// WaitEventType: (\w+)$`)
	enumValueLine := regexp.MustCompile(`^WAIT_EVENT_\w+ = (\d+);`)

	for _, line := range strings.Split(string(protoBytes), "\n") {
		line = strings.TrimSpace(line)

		if !inWaitEventEnum {
			if strings.HasPrefix(line, "enum WaitEvent {") {
				inWaitEventEnum = true
			}
			continue
		}

		if line == "}" {
			break
		}

		if match := sectionMarker.FindStringSubmatch(line); match != nil {
			currentType = match[1]
			continue
		}

		match := enumValueLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		value, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatalf("invalid enum value %q: %s", match[1], err)
		}
		if value == 0 {
			continue
		}
		if _, ok := byValue[value]; ok {
			t.Fatalf("wait event value %d defined twice in %s", value, protoPath)
		}
		if currentType == "" {
			t.Fatalf("%s has value %d before any '// WaitEventType:' marker", protoPath, value)
		}
		byValue[value] = currentType
	}

	if !inWaitEventEnum {
		t.Fatalf("enum WaitEvent not found in %s", protoPath)
	}

	return byValue
}

// protoDisplayNamesByValue extracts the display names listed in the trailing
// comments of each WaitEvent enum line in the proto (e.g. "ProcArrayLock",
// or "speculative token, spectoken" for lines with several names).
func protoDisplayNamesByValue(t *testing.T) map[int][]string {
	t.Helper()

	protoBytes, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("failed to read %s: %s", protoPath, err)
	}

	byValue := map[int][]string{}
	enumValueLine := regexp.MustCompile(`^WAIT_EVENT_\w+ = \d+; // (.+)$`)
	parenRe := regexp.MustCompile(`\([^)]*\)`)

	inWaitEventEnum := false
	for _, line := range strings.Split(string(protoBytes), "\n") {
		line = strings.TrimSpace(line)

		if !inWaitEventEnum {
			if strings.HasPrefix(line, "enum WaitEvent {") {
				inWaitEventEnum = true
			}
			continue
		}
		if line == "}" {
			break
		}

		match := enumValueLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		value := enumValueFromLine(t, line)
		if value == 0 {
			continue
		}

		// Comments look like "SyncRepLock (The display name has duplicate
		// ...)" or "CLogControlLock, XactSLRULock (moved to ...)"; drop
		// parenthetical annotations, then split the remaining names.
		comment := parenRe.ReplaceAllString(match[1], "")
		for _, name := range strings.Split(comment, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			byValue[value] = append(byValue[value], name)
		}
	}

	return byValue
}

func enumValueFromLine(t *testing.T, line string) int {
	t.Helper()

	match := regexp.MustCompile(`^WAIT_EVENT_\w+ = (\d+);`).FindStringSubmatch(line)
	if match == nil {
		t.Fatalf("unexpected enum line %q", line)
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("invalid enum value in %q: %s", line, err)
	}
	return value
}

// marshalReverseMapping formats the map with keys sorted numerically so the
// generated file is stable across runs.
func marshalReverseMapping(byValue map[int]string) []byte {
	values := make([]int, 0, len(byValue))
	for value := range byValue {
		values = append(values, value)
	}
	sort.Ints(values)

	var sb strings.Builder
	sb.WriteString("{\n")
	for i, value := range values {
		comma := ","
		if i == len(values)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, "  %q: %q%s\n", strconv.Itoa(value), byValue[value], comma)
	}
	sb.WriteString("}\n")

	return []byte(sb.String())
}

func TestWaitEventTypeOfEventJSON(t *testing.T) {
	generated := marshalReverseMapping(waitEventTypeByValue(t))

	if *update {
		if err := os.WriteFile(reverseMappingPath, generated, 0o644); err != nil {
			t.Fatalf("failed to write %s: %s", reverseMappingPath, err)
		}
		t.Logf("regenerated %s", reverseMappingPath)
		return
	}

	committed, err := os.ReadFile(reverseMappingPath)
	if err != nil {
		t.Fatalf("failed to read %s: %s (run make generate_wait_event_mappings to create it)", reverseMappingPath, err)
	}

	if string(committed) != string(generated) {
		t.Fatalf("%s is out of date with %s (run make generate_wait_event_mappings and commit the result)", reverseMappingPath, protoPath)
	}
}

// loadJSONMap reads a JSON object into the given target.
func loadJSONMap(t *testing.T, path string, target any) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %s", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to parse %s: %s", path, err)
	}
}

// TestWaitEventMappingFiles checks that the forward mapping (wait_event.json),
// the resolver (wait_event_type_resolver.json) and the reverse mapping agree
// with each other and with the proto enum.
func TestWaitEventMappingFiles(t *testing.T) {
	byValue := waitEventTypeByValue(t)

	var waitEvents map[string]int
	loadJSONMap(t, waitEventNamesMappingPath, &waitEvents)

	var waitEventTypes map[string]int
	loadJSONMap(t, waitEventTypeMappingPath, &waitEventTypes)

	// Every display name in the proto comments must map to its enum value
	// in wait_event.json. This catches the two files drifting apart, which
	// otherwise silently produces IDs that disagree between the schema docs
	// and the values the app stores.
	for value, names := range protoDisplayNamesByValue(t) {
		for _, name := range names {
			mappedValue, ok := waitEvents[name]
			if !ok {
				t.Errorf("%s lists wait event name %q for value %d but %s has no such key", protoPath, name, value, waitEventNamesMappingPath)
				continue
			}
			if mappedValue != value {
				t.Errorf("%s maps %q to %d but %s defines it as %d", waitEventNamesMappingPath, name, mappedValue, protoPath, value)
			}
		}
	}
	var resolver map[string]map[string]int
	loadJSONMap(t, waitEventTypeResolverPath, &resolver)

	var reverse map[string]string
	loadJSONMap(t, reverseMappingPath, &reverse)

	for name, typeToValue := range resolver {
		if len(typeToValue) < 2 {
			t.Errorf("%s entry %q has a single type; remove it, the flat wait_event.json lookup covers it", waitEventTypeResolverPath, name)
		}

		canonicalTypes := map[int]bool{}
		for typeName, value := range typeToValue {
			if _, ok := waitEventTypes[typeName]; !ok {
				t.Errorf("resolver %s references wait event type %q which is not in %s", name, typeName, waitEventTypeMappingPath)
			}
			canonicalType, ok := byValue[value]
			if !ok {
				t.Errorf("resolver %s -> %s points to undefined wait event value %d", name, typeName, value)
				continue
			}
			if canonicalType != typeName {
				t.Errorf("resolver %s -> %s (%d) conflicts with the proto, where %d is under type %s", name, typeName, value, value, canonicalType)
			}
			canonicalTypes[value] = true
		}
		if len(canonicalTypes) < len(typeToValue) {
			t.Errorf("resolver %s maps several types to the same enum value", name)
		}

		// If wait_event.json still carries the name (it does for SyncRep,
		// so old snapshots resolve), its flat value must be one of the
		// resolver values, so the two lookup paths cannot disagree.
		if flatValue, ok := waitEvents[name]; ok {
			if !canonicalTypes[flatValue] {
				t.Errorf("%s maps %q to %d, which is not one of the resolver values for that name", waitEventNamesMappingPath, name, flatValue)
			}
		}
	}

	// Any value used in the forward mapping must be a defined enum value
	// with exactly one canonical type, unless the name is resolver-covered.
	valuesByType := map[string]map[string]int{}
	for name, value := range waitEvents {
		if value == 0 {
			continue
		}
		canonicalType, ok := byValue[value]
		if !ok {
			t.Errorf("%s maps %q to %d, which is not a wait event value in the proto", waitEventNamesMappingPath, name, value)
			continue
		}
		if valuesByType[name] == nil {
			valuesByType[name] = map[string]int{}
		}
		valuesByType[name][canonicalType] = value
	}

	for name, types := range valuesByType {
		if len(types) > 1 {
			if _, ok := resolver[name]; !ok {
				t.Errorf("wait event name %q maps to values of different types %v, add it to %s", name, types, waitEventTypeResolverPath)
			}
		}
	}

	// Every reverse mapping entry must agree with the proto-derived map.
	for valueString, typeName := range reverse {
		value, err := strconv.Atoi(valueString)
		if err != nil {
			t.Errorf("%s has non-numeric key %q", reverseMappingPath, valueString)
			continue
		}
		if canonicalType, ok := byValue[value]; !ok {
			t.Errorf("%s references undefined wait event value %d", reverseMappingPath, value)
		} else if canonicalType != typeName {
			t.Errorf("%s says %d is type %q, but the proto puts it under %q", reverseMappingPath, value, typeName, canonicalType)
		}
	}
}

// waitEventNamesBySection parses a wait_event_names.txt and returns the
// display names Postgres shows in pg_stat_activity, grouped by wait event
// type. Names in the LWLock and Lock sections are literal display names
// (those types have hand-written C code); every other section's names are
// derived from the enum constant by capitalizing each underscore-separated
// word.
func waitEventNamesBySection(content string) (map[string]map[string]bool, error) {
	const sectionPrefix = "Section: ClassName - "

	literalNameSections := map[string]bool{"WaitEventLWLock": true, "WaitEventLock": true}

	byType := map[string]map[string]bool{}
	currentSection := ""

	for _, line := range strings.Split(content, "\n") {
		if section, ok := strings.CutPrefix(line, sectionPrefix); ok {
			currentSection = strings.TrimSpace(section)
			continue
		}
		if currentSection == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "ABI_compatibility:") {
			continue
		}

		name, _, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		displayName := name
		if !literalNameSections[currentSection] {
			displayName = toCamelCase(name)
		}

		typeName := strings.TrimPrefix(currentSection, "WaitEvent")
		if byType[typeName] == nil {
			byType[typeName] = map[string]bool{}
		}
		byType[typeName][displayName] = true
	}

	if len(byType) == 0 {
		return nil, fmt.Errorf("no wait event sections found in wait_event_names.txt")
	}

	return byType, nil
}

func toCamelCase(constant string) string {
	parts := strings.Split(constant, "_")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, strings.ToUpper(part[:1])+strings.ToLower(part[1:]))
	}
	return strings.Join(out, "")
}

// fetchText tries each path on raw.githubusercontent.com and returns the
// first that exists.
func fetchText(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func fetchWaitEventNames(branch string) (string, error) {
	return fetchText("https://raw.githubusercontent.com/postgres/postgres/" + branch + "/" + upstreamNamesFile)
}

// TestPostgresWaitEventNamesAreUnique verifies the invariant the reverse
// mapping depends on: in every supported Postgres version, a wait event name
// selects exactly one wait event type in pg_stat_activity. Names that Postgres
// reports under two types (today only SyncRep, LWLock and IPC) must appear in
// wait_event_type_resolver.json.
func TestPostgresWaitEventNamesAreUnique(t *testing.T) {
	if os.Getenv("PGANALYZE_CHECK_POSTGRES_WAIT_EVENT_NAMES") == "" {
		t.Skip("set PGANALYZE_CHECK_POSTGRES_WAIT_EVENT_NAMES=1 to check against upstream wait_event_names.txt")
	}
	if testing.Short() {
		t.Skip("skipping network-bound check in short mode")
	}

	var resolver map[string]map[string]int
	loadJSONMap(t, waitEventTypeResolverPath, &resolver)

	var (
		parsed  int
		covered = map[string]bool{}
	)

	for _, branch := range requiredPostgresBranches {
		content, err := fetchWaitEventNames(branch)
		if err != nil {
			t.Errorf("failed to fetch wait event names for %s: %s", branch, err)
			continue
		}
		parsed++

		byType, err := waitEventNamesBySection(content)
		if err != nil {
			t.Errorf("failed to parse wait event names for %s: %s", branch, err)
			continue
		}

		typesByName := map[string][]string{}
		for typeName, names := range byType {
			for name := range names {
				typesByName[name] = append(typesByName[name], typeName)
			}
		}

		for _, name := range sortedKeys(typesByName) {
			types := typesByName[name]
			if len(types) < 2 {
				continue
			}
			sort.Strings(types)

			entries, ok := resolver[name]
			if !ok {
				t.Errorf("%s reports wait event %q under multiple types %v without a resolver entry, add it to %s", branch, name, types, waitEventTypeResolverPath)
				continue
			}
			covered[name] = true
			for _, typeName := range types {
				if _, ok := entries[typeName]; !ok {
					t.Errorf("%s reports wait event %q under type %s but %s has no entry for that type", branch, name, typeName, waitEventTypeResolverPath)
				}
			}
		}
	}

	if parsed == 0 {
		t.Error("no upstream wait_event_names.txt could be fetched, the uniqueness check did not run")
		return
	}

	for _, name := range sortedKeys(resolver) {
		if !covered[name] {
			t.Logf("resolver entry %q does not collide in the Postgres branches we fetched; it may be unnecessary", name)
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
