package snapshot

import (
	"github.com/tinylib/msgp/msgp"
	"gopkg.in/guregu/null.v2"
)

type NullableString null.String

func (n NullableString) MarshalMsg(b []byte) ([]byte, error) {
	if !n.Valid {
		return msgp.AppendNil(b), nil
	}
	return msgp.AppendString(b, n.String), nil
}

func (n *NullableString) UnmarshalMsg(b []byte) ([]byte, error) {
	if msgp.IsNil(b) {
		*n = NullableString(null.StringFromPtr(nil))
		return msgp.ReadNilBytes(b)
	}
	s, o, err := msgp.ReadStringBytes(b)
	if err != nil {
		return nil, err
	}
	*n = NullableString(null.StringFrom(s))
	return o, nil
}

func (s *NullableString) EncodeMsg(w *msgp.Writer) error {
	if s.Valid {
		return w.WriteString(s.String)
	}
	return w.WriteNil()
}

func (s *NullableString) DecodeMsg(r *msgp.Reader) error {
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
		*s = NullableString(null.StringFromPtr(nil))
		return nil
	case msgp.StrType:
		str, err := r.ReadString()
		if err != nil {
			return err
		}
		*s = NullableString(null.StringFrom(str))
		return nil
	default:
		return msgp.TypeError{Encoded: typ, Method: msgp.StrType}
	}
}

func (s *NullableString) Msgsize() int {
	if s.Valid {
		return msgp.StringPrefixSize + len(s.String)
	}
	return 1
}
