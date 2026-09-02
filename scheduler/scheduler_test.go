package scheduler

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/pganalyze/collector/util"
)

func TestScheduler(t *testing.T) {
	scheduler, err := GetScheduler()
	if err != nil {
		t.Errorf("Error: %v\n", err)
	}

	someTime := time.Date(2013, 1, 1, 0, 5, 0, 0, time.UTC)
	expectedNextRun := time.Date(2013, 1, 1, 0, 10, 0, 0, time.UTC)
	actualNextRun := scheduler.TenMinute.interval.Next(someTime)

	if expectedNextRun != actualNextRun {
		t.Errorf("\nNext run:\n\texpected %s\n\tactual %s\n\n", expectedNextRun, actualNextRun)
	}
}

// Verifies that scheduled runs get a deadline that ends right before the next
// scheduled execution, so a stuck run cannot overlap the run after it. Without a
// deadline on the secondary schedule, a wedged 1-minute stats run held
// HighFreqStateMutex indefinitely, which blocks the full snapshot (mutex
// acquisition ignores context cancellation) and the collector's reload.
func TestRunWithDeadline(t *testing.T) {
	thisExecution := time.Date(2013, 1, 1, 0, 1, 0, 0, time.UTC)

	tests := []struct {
		name          string
		nextExecution time.Time
		expected      time.Time
	}{
		{
			name:          "one minute interval ends before the next execution",
			nextExecution: thisExecution.Add(1 * time.Minute),
			expected:      thisExecution.Add(59 * time.Second),
		},
		{
			name:          "ten second interval is extended to avoid pointless cancellations",
			nextExecution: thisExecution.Add(10 * time.Second),
			expected:      thisExecution.Add(19 * time.Second),
		},
	}

	for _, test := range tests {
		var actual time.Time
		var ok bool
		runWithDeadline(context.Background(), func(ctx context.Context) {
			actual, ok = ctx.Deadline()
		}, thisExecution, test.nextExecution)

		if !ok {
			t.Errorf("%s:\n\texpected a deadline on the runner's context, got none\n", test.name)
		} else if actual != test.expected {
			t.Errorf("%s:\n\texpected %s\n\tactual %s\n\n", test.name, test.expected, actual)
		}
	}
}

// Verifies that ScheduleSecondary actually applies that deadline
func TestScheduleSecondaryRunsHaveDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	// A two second secondary schedule, with a primary that only runs on the hour,
	// so no run gets skipped for overlapping with the primary
	secondary := Schedule{interval: cronexpr.MustParse("*/2 * * * * * *")}
	primary := Schedule{interval: cronexpr.MustParse("0 0 * * * * *")}

	deadlines := make(chan time.Time, 1)
	secondary.ScheduleSecondary(ctx, primary, &wg, func(ctx context.Context) {
		deadline, _ := ctx.Deadline()
		select {
		case deadlines <- deadline:
		default:
		}
	}, &util.Logger{Destination: log.New(io.Discard, "", 0)}, "test runs")

	select {
	case deadline := <-deadlines:
		if deadline.IsZero() {
			t.Error("TestScheduleSecondaryRunsHaveDeadline: expected the runner's context to carry a deadline, got none")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("TestScheduleSecondaryRunsHaveDeadline: secondary schedule did not run")
	}

	cancel()
	wg.Wait()
}
