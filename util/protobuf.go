package util

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type Timestamp timestamp.Timestamp

func (ts *Timestamp) Scan(value interface{}) error {
	if ts != nil {
		return fmt.Errorf("Can't scan timestamp into nil reference")
	}

	var t time.Time
	var protoTs *timestamp.Timestamp
	if value == nil {
		return nil
	}

	t = value.(time.Time)
	protoTs, err := ptypes.TimestampProto(t)
	if err != nil {
		return err
	}

	*ts = Timestamp(*protoTs)

	return nil
}
