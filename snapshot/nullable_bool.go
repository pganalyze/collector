package snapshot

import (
	"github.com/tinylib/msgp/msgp"
	"gopkg.in/guregu/null.v2"
)

type NullableBool null.Bool

func (val NullableBool) MarshalMsg(b []byte) ([]byte, error) {
	if !val.Valid {
		return msgp.AppendNil(b), nil
	}
	return msgp.AppendBool(b, val.Bool), nil
}

func (val *NullableBool) UnmarshalMsg(b []byte) ([]byte, error) {
	if msgp.IsNil(b) {
		*val = NullableBool(null.BoolFromPtr(nil))
		return msgp.ReadNilBytes(b)
	}
	bl, o, err := msgp.ReadBoolBytes(b)
	if err != nil {
		return nil, err
	}
	*val = NullableBool(null.BoolFrom(bl))
	return o, nil
}

func (val *NullableBool) EncodeMsg(w *msgp.Writer) error {
	if val.Valid {
		return w.WriteBool(val.Bool)
	}
	return w.WriteNil()
}

func (val *NullableBool) DecodeMsg(r *msgp.Reader) error {
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
		*val = NullableBool(null.BoolFromPtr(nil))
		return nil
	case msgp.BoolType:
		bl, err := r.ReadBool()
		if err != nil {
			return err
		}
		*val = NullableBool(null.BoolFrom(bl))
		return nil
	default:
		return msgp.TypeError{Encoded: typ, Method: msgp.BoolType}
	}
}

func (val *NullableBool) Msgsize() int {
	return 1
}
