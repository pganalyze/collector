package pganalyze_collector

import (
	"github.com/golang/protobuf/ptypes"
	"gopkg.in/guregu/null.v3"
)

func NullTimeToNullTimestamp(in null.Time) *NullTimestamp {
	if !in.Valid {
		return &NullTimestamp{Valid: false}
	}

	ts, _ := ptypes.TimestampProto(in.Time)

	return &NullTimestamp{Valid: true, Value: ts}
}
