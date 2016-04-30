package snapshot

import (
	"github.com/tinylib/msgp/msgp"
	"gopkg.in/guregu/null.v2"
)

type NullableInt null.Int

func (val NullableInt) MarshalMsg(b []byte) ([]byte, error) {
	if !val.Valid {
		return msgp.AppendNil(b), nil
	}
	return msgp.AppendInt64(b, val.Int64), nil
}

func (val *NullableInt) UnmarshalMsg(b []byte) ([]byte, error) {
	if msgp.IsNil(b) {
		*val = NullableInt(null.IntFromPtr(nil))
		return msgp.ReadNilBytes(b)
	}
	i, o, err := msgp.ReadInt64Bytes(b)
	if err != nil {
		return nil, err
	}
	*val = NullableInt(null.IntFrom(i))
	return o, nil
}

func (val *NullableInt) EncodeMsg(w *msgp.Writer) error {
	if val.Valid {
		return w.WriteInt64(val.Int64)
	}
	return w.WriteNil()
}

func (val *NullableInt) DecodeMsg(r *msgp.Reader) error {
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
		*val = NullableInt(null.IntFromPtr(nil))
		return nil
	case msgp.IntType:
		i, err := r.ReadInt64()
		if err != nil {
			return err
		}
		*val = NullableInt(null.IntFrom(i))
		return nil
	default:
		return msgp.TypeError{Encoded: typ, Method: msgp.IntType}
	}
}

func (val *NullableInt) Msgsize() int {
	return msgp.Int64Size
}
