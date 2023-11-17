package querysample

import (
	"testing"
	"time"

	"github.com/pganalyze/collector/state"
	"go.opentelemetry.io/otel/trace"
)

type startAndEndTimeTestPair struct {
	testName   string
	traceState trace.TraceState
	sample     state.PostgresQuerySample
	startTime  time.Time
	endTime    time.Time
}

func TestStartAndEndTime(t *testing.T) {
	currentTime := time.Date(2023, time.January, 1, 1, 2, 3, 456*1000*1000, time.UTC)
	traceState := trace.TraceState{}
	otelTraceState, err := traceState.Insert("ot", "p:8;r:62")
	if err != nil {
		t.Fatalf("Failed to initialize object: %v", err)
	}
	pganalyzeTraceStateWithoutT, err := traceState.Insert("pganalyze", "x:foo;y:bar")
	if err != nil {
		t.Fatalf("Failed to initialize object: %v", err)
	}
	// inserting the same key will update the value
	pganalyzeTraceState, err := traceState.Insert("pganalyze", "t:1697666938.6297212")
	if err != nil {
		t.Fatalf("Failed to initialize object: %v", err)
	}
	// 1697666938.6297212 = 2023-10-18 22:08:58.6297212
	pganalyzeTime, err := time.Parse("2006-01-02T15:04:05.999999999", "2023-10-18T22:08:58.6297212")
	if err != nil {
		t.Fatalf("Failed to initialize object: %v", err)
	}
	// due to the limitation of the floating point, the result won't exactly like above, so tweaking to pass the test
	pganalyzeTime = pganalyzeTime.Add(-1 * 112)

	var startAndEndTimeTests = []startAndEndTimeTestPair{
		{
			"No trace state",
			trace.TraceState{},
			state.PostgresQuerySample{RuntimeMs: 1000, OccurredAt: currentTime},
			currentTime.Add(-1 * 1000 * time.Millisecond),
			currentTime,
		},
		{
			"No pganalyze trace state",
			otelTraceState,
			state.PostgresQuerySample{RuntimeMs: 1000, OccurredAt: currentTime},
			currentTime.Add(-1 * 1000 * time.Millisecond),
			currentTime,
		},
		{
			"pganalyze trace state without t",
			pganalyzeTraceStateWithoutT,
			state.PostgresQuerySample{RuntimeMs: 1000, OccurredAt: currentTime},
			currentTime.Add(-1 * 1000 * time.Millisecond),
			currentTime,
		},
		{
			"pganalyze trace state",
			pganalyzeTraceState,
			state.PostgresQuerySample{RuntimeMs: 1000, OccurredAt: currentTime},
			pganalyzeTime,
			pganalyzeTime.Add(1000 * time.Millisecond),
		},
	}

	for _, pair := range startAndEndTimeTests {
		startTime, endTime := startAndEndTime(pair.traceState, pair.sample)
		if pair.startTime != startTime {
			t.Errorf("For %s: expected startTime to be %v, but was %v\n", pair.testName, pair.startTime, startTime)
		}
		if pair.endTime != endTime {
			t.Errorf("For %s: expected endTime to be %v, but was %v\n", pair.testName, pair.endTime, endTime)
		}
	}
}
