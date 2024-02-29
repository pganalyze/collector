package scheduler

import (
	"context"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/pganalyze/collector/util"
)

type Group struct {
	interval *cronexpr.Expression
}

func (group Group) Schedule(ctx context.Context, runner func(context.Context), logger *util.Logger, logName string) {
	go func() {
		for {
			nextExecutions := group.interval.NextN(time.Now(), 2)
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
// where the primary group also has a run (to avoid overlapping statistics)
func (group Group) ScheduleSecondary(ctx context.Context, runner func(context.Context), logger *util.Logger, logName string, primaryGroup Group) {
	go func() {
		for {
			timeNow := time.Now()
			delay := group.interval.Next(timeNow).Sub(timeNow)
			delayPrimary := primaryGroup.interval.Next(timeNow).Sub(timeNow)

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
					logger.PrintVerbose("Skipping run for %s since it overlaps with primary group time", logName)
				} else {
					runner(ctx)
				}
			}
		}
	}()
}

const FullSnapshotsPerHour = 6

func GetSchedulerGroups() (groups map[string]Group, err error) {
	tenSecondInterval, err := cronexpr.Parse("*/10 * * * * * *")
	if err != nil {
		return
	}

	oneMinuteInterval, err := cronexpr.Parse("0 * * * * * *")
	if err != nil {
		return
	}

	// TODO(ianstanton) For local dev. Revert this change.
	tenMinuteInterval, err := cronexpr.Parse("0 */1 * * * * *")
	if err != nil {
		return
	}

	groups = make(map[string]Group)

	groups["stats"] = Group{interval: tenMinuteInterval}
	groups["reports"] = Group{interval: oneMinuteInterval}
	groups["activity"] = Group{interval: tenSecondInterval}
	groups["query_stats"] = Group{interval: oneMinuteInterval}

	return
}
