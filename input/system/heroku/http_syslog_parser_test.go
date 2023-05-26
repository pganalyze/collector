package heroku_test

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pganalyze/collector/input/system/heroku"
)

type logplexTestpair struct {
	in  string
	out []heroku.HttpSyslogMessage
}

var logplexTests = []logplexTestpair{
	{
		`688 <134>1 2023-06-05T19:27:18+00:00 host app heroku-postgres - source=HEROKU_POSTGRESQL_YELLOW addon=postgresql-reticulated-12345 sample#current_transaction=19028263 sample#db_size=10311979679bytes sample#tables=1443 sample#active-connections=11 sample#waiting-connections=0 sample#index-cache-hit-rate=0.99814 sample#table-cache-hit-rate=0.96729 sample#load-avg-1m=0 sample#load-avg-5m=0 sample#load-avg-15m=0 sample#read-iops=0 sample#write-iops=0.21311 sample#tmp-disk-used=543600640 sample#tmp-disk-available=72435191808 sample#memory-total=8036340kB sample#memory-free=80640kB sample#memory-cached=7319340kB sample#memory-postgres=23380kB sample#wal-percentage-used=0.06427136779541504` + "\n",
		[]heroku.HttpSyslogMessage{
			{
				HeaderTimestamp: "2023-06-05T19:27:18+00:00",
				HeaderProcID:    "heroku-postgres",
				Content:         []byte(`source=HEROKU_POSTGRESQL_YELLOW addon=postgresql-reticulated-12345 sample#current_transaction=19028263 sample#db_size=10311979679bytes sample#tables=1443 sample#active-connections=11 sample#waiting-connections=0 sample#index-cache-hit-rate=0.99814 sample#table-cache-hit-rate=0.96729 sample#load-avg-1m=0 sample#load-avg-5m=0 sample#load-avg-15m=0 sample#read-iops=0 sample#write-iops=0.21311 sample#tmp-disk-used=543600640 sample#tmp-disk-available=72435191808 sample#memory-total=8036340kB sample#memory-free=80640kB sample#memory-cached=7319340kB sample#memory-postgres=23380kB sample#wal-percentage-used=0.06427136779541504` + "\n"),
				Path:            "",
			},
		},
	},
	{
		`451 <134>1 2023-06-05T19:33:14+00:00 host app postgres.1105947 - [YELLOW] [12-1]  sql_error_code = 00000 time_ms = "2023-06-05 19:33:14.793 UTC" pid="1991397" proc_start_time="2023-06-05 19:33:12 UTC" session_id="647e3878.1e62e5" vtid="2/0" tid="0" log_line="3" database="databasedataba" connection_source="11.222.333.444(12345)" user="useruseruserus" application_name="pganalyze_test_run" LOG:  duration: 2405.887 ms  statement: WITH naptime(value) AS (` + "\n" +
			`85 <134>1 2023-06-05T19:33:14+00:00 host app postgres.1105947 - [YELLOW] [12-2] 	SELECT` + "\n",
		[]heroku.HttpSyslogMessage{
			{
				HeaderTimestamp: "2023-06-05T19:33:14+00:00",
				HeaderProcID:    "postgres.1105947",
				Content:         []byte(`[YELLOW] [12-1]  sql_error_code = 00000 time_ms = "2023-06-05 19:33:14.793 UTC" pid="1991397" proc_start_time="2023-06-05 19:33:12 UTC" session_id="647e3878.1e62e5" vtid="2/0" tid="0" log_line="3" database="databasedataba" connection_source="11.222.333.444(12345)" user="useruseruserus" application_name="pganalyze_test_run" LOG:  duration: 2405.887 ms  statement: WITH naptime(value) AS (` + "\n"),
				Path:            "",
			},
			{
				HeaderTimestamp: "2023-06-05T19:33:14+00:00",
				HeaderProcID:    "postgres.1105947",
				Content:         []byte("[YELLOW] [12-2] 	SELECT" + "\n"),
				Path:            "",
			},
		},
	},
	// Non-Postgres message
	{
		`408 <134>1 2023-06-05T19:35:51+00:00 host app heroku-redis - source=REDIS addon=redis-closed-12345 sample#active-connections=1 sample#load-avg-1m=0.145 sample#load-avg-5m=0.145 sample#load-avg-15m=0.155 sample#read-iops=0 sample#write-iops=0 sample#memory-total=16082928kB sample#memory-free=11453204kB sample#memory-cached=2528244kB sample#memory-redis=460192bytes sample#hit-rate=0.92675 sample#evicted-keys=0` + "\n",
		[]heroku.HttpSyslogMessage{},
	},
}

func TestReadPostgresLogStreamItems(t *testing.T) {
	for _, pair := range logplexTests {
		out := heroku.ReadHerokuPostgresSyslogMessages(strings.NewReader(pair.in))

		cfg := pretty.CompareConfig
		cfg.SkipZeroFields = true
		if diff := cfg.Compare(out, pair.out); diff != "" {
			t.Errorf("For \"%v\": diff: (-got +want)\n%s", pair.in, diff)
		}
	}
}
