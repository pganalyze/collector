package scheduler

var DefaultConfig = `
[intervals]
Hourly = "0 0 * * * * *"
10min = "0 */10 * * * * *"
1min = "0 * * * * * *"

[groups]
	[groups.stats]
	Method = "fixed"
	Interval = "10min"
`
