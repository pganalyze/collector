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

func GetSchedulerGroups() (groups map[string]Group, err error) {
	oneMinuteInterval, err := cronexpr.Parse("0 * * * * * *")
	tenMinuteInterval, err := cronexpr.Parse("0 */10 * * * * *")
	if err != nil {
		return
	}

	groups = make(map[string]Group)

	groups["stats"] = Group{interval: tenMinuteInterval}
	groups["reports"] = Group{interval: oneMinuteInterval}
	groups["logs"] = Group{interval: oneMinuteInterval}

	return
}
