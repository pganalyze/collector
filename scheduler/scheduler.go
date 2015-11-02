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
			// NOTE(LukasFittl): In the future we'll measure runner's exection time
			// and decide scheduling interval based on that
			runner()

			delay := group.interval.Next(time.Now()).Sub(time.Now())

			select {
			case <-time.After(delay):
				// Nothing, re-run loop
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
