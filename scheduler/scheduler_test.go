package scheduler

import (
	"testing"
	"time"
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
