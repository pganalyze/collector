package scheduler

import (
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/pganalyze/collector/util"
)

type Group struct {
	interval *cronexpr.Expression
}

func (group Group) Schedule(runner func(), logger *util.Logger, logName string) chan bool {
	stop := make(chan bool)
	go func() {
		for {
			delay := group.interval.Next(time.Now()).Sub(time.Now())

			logger.PrintVerbose("Scheduled next run for %s in %+v", logName, delay)

			select {
			case <-time.After(delay):
				// NOTE: In the future we'll measure the runner's execution time
				// and decide the next scheduling interval based on that
				runner()
			case <-stop:
				return
			}
		}
	}()
	return stop
}

// ScheduleSecondary - Behaves almost like Schedule, but ignores the point in time
// where the primary group also has a run (to avoid overlapping statistics)
func (group Group) ScheduleSecondary(runner func(), logger *util.Logger, logName string, primaryGroup Group) chan bool {
	stop := make(chan bool)
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
			case <-time.After(delay):
				if int(delay.Seconds()) == int(delayPrimary.Seconds()) {
					logger.PrintVerbose("Skipping run for %s since it overlaps with primary group time", logName)
				} else {
					runner()
				}
			case <-stop:
				return
			}
		}
	}()
	return stop
}

func GetSchedulerGroups() (groups map[string]Group, err error) {
	tenSecondInterval, err := cronexpr.Parse("*/10 * * * * * *")
	if err != nil {
		return
	}

	thirtySecondInterval, err := cronexpr.Parse("*/30 * * * * * *")
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

	groups = make(map[string]Group)

	groups["stats"] = Group{interval: tenMinuteInterval}
	groups["reports"] = Group{interval: oneMinuteInterval}
	groups["logs"] = Group{interval: thirtySecondInterval}
	groups["activity"] = Group{interval: tenSecondInterval}
	groups["query_stats"] = Group{interval: oneMinuteInterval}

	return
}
