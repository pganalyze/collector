package scheduler

import (
	"context"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/pganalyze/collector/util"
)

type Scheduler struct {
	TenSecond Schedule
	OneMinute Schedule
	TenMinute Schedule
}

func GetScheduler() (scheduler Scheduler, err error) {
	tenSecondInterval, err := cronexpr.Parse("*/10 * * * * * *")
	if err != nil {
		return
	}

	oneMinuteInterval, err := cronexpr.Parse("0 * * * * * *")
	if err != nil {
		return
	}

	tenMinuteInterval, err := cronexpr.Parse("0 */10 * * * * *")
	if err != nil {
		return
	}

	scheduler = Scheduler{
		TenSecond: Schedule{interval: tenSecondInterval},
		OneMinute: Schedule{interval: oneMinuteInterval},
		TenMinute: Schedule{interval: tenMinuteInterval},
	}
	return
}

type Schedule struct {
	interval *cronexpr.Expression
}

func (schedule Schedule) Schedule(ctx context.Context, runner func(context.Context), logger *util.Logger, logName string) {
	go func() {
		for {
			nextExecutions := schedule.interval.NextN(time.Now(), 2)
			delay := time.Until(nextExecutions[0])

			logger.PrintVerbose("Scheduled next run for %s in %+v", logName, delay)

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				func() {
					// Cancel runner at latest right before next scheduled execution should
					// occur, to prevent skipping over runner executions by accident.
					deadline := nextExecutions[1].Add(-1 * time.Second)
					// Extend the deadline of very short runs to avoid pointless cancellations.
					if nextExecutions[1].Sub(nextExecutions[0]) < 19*time.Second {
						deadline = nextExecutions[0].Add(19 * time.Second)
					}
					ctx, cancel := context.WithDeadline(ctx, deadline)
					defer cancel()
					runner(ctx)
				}()
			}
		}
	}()
}

// ScheduleSecondary - Behaves almost like Schedule, but ignores the point in time
// where the primary schedule also has a run (to avoid overlapping statistics)
func (schedule Schedule) ScheduleSecondary(ctx context.Context, primarySchedule Schedule, runner func(context.Context), logger *util.Logger, logName string) {
	go func() {
		for {
			timeNow := time.Now()
			delay := schedule.interval.Next(timeNow).Sub(timeNow)
			delayPrimary := primarySchedule.interval.Next(timeNow).Sub(timeNow)

			// Make sure to not run more often than once a second - this can happen
			// due to rounding errors in the interval logic
			if delay.Seconds() < 1.0 {
				time.Sleep(delay)
				continue
			}

			logger.PrintVerbose("Scheduled next run for %s in %+v", logName, delay)

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				if int(delay.Seconds()) == int(delayPrimary.Seconds()) {
					logger.PrintVerbose("Skipping run for %s since it overlaps with primary schedule", logName)
				} else {
					runner(ctx)
				}
			}
		}
	}()
}

const FullSnapshotsPerHour = 6
