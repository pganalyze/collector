package snapshot

import (
	"github.com/tinylib/msgp/msgp"
	"gopkg.in/guregu/null.v2"
)

type NullableFloat null.Float

func (val NullableFloat) MarshalMsg(b []byte) ([]byte, error) {
	if !val.Valid {
		return msgp.AppendNil(b), nil
	}
	return msgp.AppendFloat64(b, val.Float64), nil
}

func (val *NullableFloat) UnmarshalMsg(b []byte) ([]byte, error) {
	if msgp.IsNil(b) {
		*val = NullableFloat(null.FloatFromPtr(nil))
		return msgp.ReadNilBytes(b)
	}
	i, o, err := msgp.ReadFloat64Bytes(b)
	if err != nil {
		return nil, err
	}
	*val = NullableFloat(null.FloatFrom(i))
	return o, nil
}

func (val *NullableFloat) EncodeMsg(w *msgp.Writer) error {
	if val.Valid {
		return w.WriteFloat64(val.Float64)
	}
	return w.WriteNil()
}

func (val *NullableFloat) DecodeMsg(r *msgp.Reader) error {
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
		*val = NullableFloat(null.FloatFromPtr(nil))
		return nil
	case msgp.IntType:
		f, err := r.ReadFloat64()
		if err != nil {
			return err
		}
		*val = NullableFloat(null.FloatFrom(f))
		return nil
	default:
		return msgp.TypeError{Encoded: typ, Method: msgp.Float64Type}
	}
}

func (val *NullableFloat) Msgsize() int {
	return msgp.Float64Size
}
