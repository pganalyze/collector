package snapshot

import (
	"time"

	"github.com/pganalyze/collector/util"
	"github.com/tinylib/msgp/msgp"
)

type NullableUnixTimestamp util.Timestamp

func (val NullableUnixTimestamp) MarshalMsg(b []byte) ([]byte, error) {
	if !val.Valid {
		return msgp.AppendNil(b), nil
	}
	return msgp.AppendInt64(b, val.Time.Unix()), nil
}

func (val *NullableUnixTimestamp) UnmarshalMsg(b []byte) ([]byte, error) {
	if msgp.IsNil(b) {
		*val = NullableUnixTimestamp(util.TimestampFromPtr(nil))
		return msgp.ReadNilBytes(b)
	}
	i, o, err := msgp.ReadInt64Bytes(b)
	if err != nil {
		return nil, err
	}
	*val = NullableUnixTimestamp(util.TimestampFrom(time.Unix(i, 0)))
	return o, nil
}

func (val *NullableUnixTimestamp) EncodeMsg(w *msgp.Writer) error {
	if val.Valid {
		return w.WriteInt64(val.Time.Unix())
	}
	return w.WriteNil()
}

func (val *NullableUnixTimestamp) DecodeMsg(r *msgp.Reader) error {
	typ, err := r.NextType()
	if err != nil {
		return err
	}
	switch typ {
	case msgp.NilType:
		err := r.ReadNil()
		if err != nil {
			return err
		}
		*val = NullableUnixTimestamp(util.TimestampFromPtr(nil))
		return nil
	case msgp.IntType:
		i, err := r.ReadInt64()
		if err != nil {
			return err
		}
		*val = NullableUnixTimestamp(util.TimestampFrom(time.Unix(i, 0)))
		return nil
	default:
		return msgp.TypeError{Encoded: typ, Method: msgp.IntType}
	}
}

func (val *NullableUnixTimestamp) Msgsize() int {
	return msgp.Int64Size
}
