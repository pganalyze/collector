//go:generate msgp

//msgp:shim null.String as:string using:NullStringToString/StringToNullString
//msgp:shim util.Timestamp as:time.Time using:NullTimestampToTime/TimeToNullTimestamp

package snapshot

import (
	"time"

	"github.com/pganalyze/collector/util"
)

type Activity struct {
	Pid             int            `msg:"pid"`
	Username        NullableString `msg:"username"`
	ApplicationName NullableString `msg:"application_name"`
	ClientAddr      NullableString `msg:"client_addr"`
	BackendStart    util.Timestamp `msg:"backend_start"`
	XactStart       util.Timestamp `msg:"xact_start"`
	QueryStart      util.Timestamp `msg:"query_start"`
	StateChange     util.Timestamp `msg:"state_change"`
	Waiting         NullableBool   `msg:"waiting"`
	State           NullableString `msg:"state"`
	NormalizedQuery NullableString `msg:"normalized_query"`
}

func TimeToNullTimestamp(t time.Time) util.Timestamp {
	return util.TimestampFrom(t)
}

func NullTimestampToTime(t util.Timestamp) time.Time {
	return t.Time
}
