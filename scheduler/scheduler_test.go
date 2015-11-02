package scheduler

import (
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	groups, err := ReadSchedulerGroups(DefaultConfig)
	if err != nil {
		t.Errorf("Error: %v\n", err)
	}

	if groups["stats"].Method != "fixed" {
		t.Errorf("Invalid method %v\n", groups["stats"].Method)
	}

	if groups["stats"].IntervalName != "10min" {
		t.Errorf("Invalid interval name %v\n", groups["stats"].IntervalName)
	}

	someTime := time.Date(2013, 1, 1, 0, 5, 0, 0, time.UTC)
	expectedNextRun := time.Date(2013, 1, 1, 0, 10, 0, 0, time.UTC)
	actualNextRun := groups["stats"].interval.Next(someTime)

	if expectedNextRun != actualNextRun {
		t.Errorf("\nNext run:\n\texpected %s\n\tactual %s\n\n", expectedNextRun, actualNextRun)
	}
}
