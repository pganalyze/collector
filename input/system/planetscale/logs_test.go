package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func TestDownloadLogFiles(t *testing.T) {
	baseTime := time.Now().Add(-1 * time.Minute)

	tests := []struct {
		name          string
		entryCount    int       // Base time gets advanced 1 ms after every entry
		lastTimestamp time.Time // zero means not set
		wantRequests  int32
		wantLogLines  int
		wantLastTime  time.Time // zero means expect zero
	}{
		{
			name:         "SinglePage",
			entryCount:   5,
			wantRequests: 1,
			wantLogLines: 5,
			wantLastTime: baseTime.Add(4 * time.Millisecond),
		},
		{
			name:          "Pagination",
			entryCount:    2500,
			lastTimestamp: baseTime.Add(-1 * time.Millisecond),
			wantRequests:  3, // 1000 + 1000 + 500
			wantLogLines:  2500,
			wantLastTime:  baseTime.Add(2499 * time.Millisecond),
		},
		{
			name:          "PaginationWithSince",
			entryCount:    1500,
			lastTimestamp: baseTime.Add(499 * time.Millisecond), // skip first 500
			wantRequests:  2,                                    // 1000 + 0
			wantLogLines:  1000,
			wantLastTime:  baseTime.Add(1499 * time.Millisecond),
		},
		{
			name:         "EmptyResponse",
			entryCount:   0,
			wantRequests: 1,
			wantLogLines: 0,
		},
		{
			name:          "ExactlyOnePage",
			entryCount:    1000,
			lastTimestamp: baseTime.Add(-1 * time.Millisecond),
			wantRequests:  2, // full first page triggers second request
			wantLogLines:  1000,
			wantLastTime:  baseTime.Add(999 * time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := generateLogEntries(tt.entryCount, baseTime, "LOG: ")
			mockServer := newMockLogsServer(entries)
			defer mockServer.Close()

			server, logger := makeTestServer(mockServer.URL)
			if !tt.lastTimestamp.IsZero() {
				server.LogPrevState.PlanetScale.LastTimestamp = tt.lastTimestamp
			}

			psl, logFiles, _, err := DownloadLogFiles(context.Background(), server, state.CollectionOpts{}, logger)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := mockServer.RequestCount(); got != tt.wantRequests {
				t.Errorf("requests: got %d, want %d", got, tt.wantRequests)
			}

			if tt.wantLogLines == 0 {
				if len(logFiles) != 0 {
					t.Errorf("log files: got %d, want 0", len(logFiles))
				}
			} else {
				if len(logFiles) != 1 {
					t.Fatalf("log files: got %d, want 1", len(logFiles))
				}
				if got := len(logFiles[0].LogLines); got != tt.wantLogLines {
					t.Errorf("log lines: got %d, want %d", got, tt.wantLogLines)
				}
			}

			if tt.wantLastTime.IsZero() {
				if !psl.PlanetScale.LastTimestamp.IsZero() {
					t.Errorf("LastTimestamp: got %v, want zero", psl.PlanetScale.LastTimestamp)
				}
			} else if !psl.PlanetScale.LastTimestamp.Equal(tt.wantLastTime) {
				t.Errorf("LastTimestamp: got %v, want %v", psl.PlanetScale.LastTimestamp, tt.wantLastTime)
			}
		})
	}
}

func TestDownloadLogFiles_SizeLimitDiscardsOlderData(t *testing.T) {
	baseTime := time.Now().Add(-1 * time.Minute)
	// 2000 entries with ~12KB messages; exceeds the 10MB buffer limit
	largeMsg := "LOG: " + strings.Repeat("x", 12*1024)
	entries := generateLogEntries(2000, baseTime, largeMsg)

	mockServer := newMockLogsServer(entries)
	defer mockServer.Close()

	server, logger := makeTestServer(mockServer.URL)
	server.LogPrevState.PlanetScale.LastTimestamp = baseTime.Add(-1 * time.Millisecond)

	psl, logFiles, _, err := DownloadLogFiles(context.Background(), server, state.CollectionOpts{}, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1000 + 1000 + 0: pagination continues despite exceeding the size limit
	if got := mockServer.RequestCount(); got != 3 {
		t.Errorf("requests: got %d, want 3", got)
	}

	// Some lines are lost due to size limit truncating older data
	if len(logFiles) != 1 {
		t.Fatalf("log files: got %d, want 1", len(logFiles))
	}
	if got := len(logFiles[0].LogLines); got == 0 || got >= 2000 {
		t.Errorf("log lines: got %d, want between 1 and 1999", got)
	}

	// LastTimestamp reflects the newest entry across all pages
	expectedTime := baseTime.Add(1999 * time.Millisecond)
	if !psl.PlanetScale.LastTimestamp.Equal(expectedTime) {
		t.Errorf("LastTimestamp: got %v, want %v", psl.PlanetScale.LastTimestamp, expectedTime)
	}
}

func TestDownloadLogFiles_MultipleCalls(t *testing.T) {
	baseTime := time.Now().Add(-1 * time.Minute)

	mockServer := newMockLogsServer(nil)
	defer mockServer.Close()

	server, logger := makeTestServer(mockServer.URL)
	ctx := context.Background()

	type callResult struct {
		psl      state.PersistedLogState
		logFiles []state.LogFile
	}

	call := func(label string) callResult {
		psl, logFiles, _, err := DownloadLogFiles(ctx, server, state.CollectionOpts{}, logger)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", label, err)
		}
		return callResult{psl, logFiles}
	}

	checkResult := func(label string, r callResult, wantRequests int32, wantLogLines int, wantLastTime time.Time) {
		if got := mockServer.RequestCount(); got != wantRequests {
			t.Errorf("%s: requests: got %d, want %d", label, got, wantRequests)
		}
		if wantLogLines == 0 {
			if len(r.logFiles) != 0 {
				t.Errorf("%s: log files: got %d, want 0", label, len(r.logFiles))
			}
		} else {
			if len(r.logFiles) != 1 {
				t.Fatalf("%s: log files: got %d, want 1", label, len(r.logFiles))
			}
			if got := len(r.logFiles[0].LogLines); got != wantLogLines {
				t.Errorf("%s: log lines: got %d, want %d", label, got, wantLogLines)
			}
		}
		if !r.psl.PlanetScale.LastTimestamp.Equal(wantLastTime) {
			t.Errorf("%s: LastTimestamp: got %v, want %v", label, r.psl.PlanetScale.LastTimestamp, wantLastTime)
		}
	}

	// Call 1: initial fetch with 500 entries, no LastTimestamp
	mockServer.Append(generateLogEntries(500, baseTime, "LOG: "))
	r := call("call 1")
	checkResult("call 1", r, 1, 500, baseTime.Add(499*time.Millisecond))

	// Call 2: 300 new entries arrive
	newBaseTime := baseTime.Add(500 * time.Millisecond)
	mockServer.Append(generateLogEntries(300, newBaseTime, "LOG: "))
	server.LogPrevState = r.psl
	mockServer.ResetRequestCount()
	r = call("call 2")
	checkResult("call 2", r, 1, 300, newBaseTime.Add(299*time.Millisecond))

	// Call 3: no new entries
	server.LogPrevState = r.psl
	mockServer.ResetRequestCount()
	prevLastTime := r.psl.PlanetScale.LastTimestamp
	r = call("call 3")
	checkResult("call 3", r, 1, 0, prevLastTime)

	// Call 4: 1500 new entries, requiring pagination
	newBaseTime2 := newBaseTime.Add(300 * time.Millisecond)
	mockServer.Append(generateLogEntries(1500, newBaseTime2, "LOG: "))
	server.LogPrevState = r.psl
	mockServer.ResetRequestCount()
	r = call("call 4")
	checkResult("call 4", r, 2, 1500, newBaseTime2.Add(1499*time.Millisecond))
}

// Test helpers

// generateLogEntries creates N log entries with timestamps starting at
// baseTime, each 1ms apart. Log messages use the PlanetScale "[POSTGRES]"
// prefix followed by msgPrefix (which should include a Postgres log level
// like "LOG: " for the parser to recognize them).
func generateLogEntries(n int, baseTime time.Time, msgPrefix string) []LogEntry {
	entries := make([]LogEntry, n)
	for i := range n {
		t := baseTime.Add(time.Duration(i) * time.Millisecond)
		entries[i] = LogEntry{
			Time:      t.Format(time.RFC3339Nano),
			StreamID:  "stream-1",
			Msg:       fmt.Sprintf("[POSTGRES] %s [1] %s entry %d", t.Format("2006-01-02 15:04:05.000 MST"), msgPrefix, i),
			Component: "postgres",
			Role:      "primary",
			BranchID:  "branch-123",
			Pod:       "pod-1",
		}
	}
	return entries
}

// parseTimeFilter extracts the timestamp from a _time:>TIMESTAMP filter.
func parseTimeFilter(query string) (time.Time, bool) {
	idx := strings.Index(query, "_time:>")
	if idx == -1 {
		return time.Time{}, false
	}
	timeStr := query[idx+len("_time:>"):]
	if spaceIdx := strings.Index(timeStr, " "); spaceIdx != -1 {
		timeStr = timeStr[:spaceIdx]
	}
	since, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return time.Time{}, false
	}
	return since, true
}

// writeEntries writes log entries as newline-delimited JSON to the response.
func writeEntries(w http.ResponseWriter, entries []LogEntry) {
	enc := json.NewEncoder(w)
	for _, e := range entries {
		enc.Encode(e)
	}
}

// mockLogsServer wraps an httptest.Server, tracks request count, and holds
// a mutable set of log entries. Entries can be added via Append between
// requests to simulate new log data arriving over time.
type mockLogsServer struct {
	*httptest.Server
	requestCount int32
	mu           sync.Mutex
	entries      []LogEntry
}

func (m *mockLogsServer) RequestCount() int32 { return atomic.LoadInt32(&m.requestCount) }
func (m *mockLogsServer) ResetRequestCount()  { atomic.StoreInt32(&m.requestCount, 0) }
func (m *mockLogsServer) Append(entries []LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entries...)
}
func (m *mockLogsServer) getEntries() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]LogEntry(nil), m.entries...)
}

func newMockLogsServer(entries []LogEntry) *mockLogsServer {
	m := &mockLogsServer{entries: entries}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&m.requestCount, 1)

		entries := m.getEntries()

		query := r.URL.Query().Get("query")
		limitStr := r.URL.Query().Get("limit")
		limit, _ := strconv.Atoi(limitStr)
		if limit == 0 {
			limit = 1000
		}

		since, hasTimeFilter := parseTimeFilter(query)
		var res []LogEntry
		for _, e := range entries {
			t, _ := time.Parse(time.RFC3339Nano, e.Time)
			if !hasTimeFilter || t.After(since) {
				res = append(res, e)
			}
			if len(res) >= limit {
				break
			}
		}
		writeEntries(w, res)
	}))
	return m
}

// makeTestServer creates a state.Server and logger configured for testing.
// BranchID, Signature and LogParser are pre-populated.
func makeTestServer(logsURL string) (*state.Server, *util.Logger) {
	server := state.MakeServer(config.ServerConfig{
		PlanetScaleLogsURL: logsURL,
		HTTPClient:         http.DefaultClient,
	}, false)
	server.LogPrevState = state.PersistedLogState{}
	server.LogPrevState.PlanetScale.BranchID = "branch-123"
	server.LogPrevState.PlanetScale.Signature = "test-sig"
	server.LogPrevState.PlanetScale.Expiry = time.Now().Add(1 * time.Hour).Unix()
	server.LogParser = logs.NewLogParser("[POSTGRES] %m [%p] ", nil)

	logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
	return server, logger
}
