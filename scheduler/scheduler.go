package scheduler

import (
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gorhill/cronexpr"
)

type config struct {
	Intervals map[string]string `toml:"intervals"`
	Groups    map[string]Group
}

type Group struct {
	Method       string
	IntervalName string `toml:"Interval"`
	interval     *cronexpr.Expression
}

func (group Group) Schedule(runner func()) chan bool {
	stop := make(chan bool)

	go func() {
		for {
			delay := group.interval.Next(time.Now()).Sub(time.Now())

			select {
			case <-time.After(delay):
				// NOTE(LukasFittl): In the future we'll measure the runner's exection time
				// and decide the next scheduling interval based on that
				runner()
			case <-stop:
				return
			}
		}
	}()

	return stop
}

func ReadSchedulerGroups(configData string) (groups map[string]Group, err error) {
	var config config
	if _, err = toml.Decode(configData, &config); err != nil {
		return
	}

	for key, group := range config.Groups {
		var expr *cronexpr.Expression
		if expr, err = cronexpr.Parse(config.Intervals[group.IntervalName]); err != nil {
			return
		}
		group.interval = expr
		config.Groups[key] = group
	}

	groups = config.Groups

	return
}
