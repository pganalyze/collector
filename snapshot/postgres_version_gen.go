package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *PostgresVersion) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var xvk uint32
	xvk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for xvk > 0 {
		xvk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "full":
			z.Full, err = dc.ReadString()
			if err != nil {
				return
			}
		case "short":
			z.Short, err = dc.ReadString()
			if err != nil {
				return
			}
		case "numeric":
			z.Numeric, err = dc.ReadInt()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z PostgresVersion) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "full"
	err = en.Append(0x83, 0xa4, 0x66, 0x75, 0x6c, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Full)
	if err != nil {
		return
	}
	// write "short"
	err = en.Append(0xa5, 0x73, 0x68, 0x6f, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Short)
	if err != nil {
		return
	}
	// write "numeric"
	err = en.Append(0xa7, 0x6e, 0x75, 0x6d, 0x65, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Numeric)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z PostgresVersion) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "full"
	o = append(o, 0x83, 0xa4, 0x66, 0x75, 0x6c, 0x6c)
	o = msgp.AppendString(o, z.Full)
	// string "short"
	o = append(o, 0xa5, 0x73, 0x68, 0x6f, 0x72, 0x74)
	o = msgp.AppendString(o, z.Short)
	// string "numeric"
	o = append(o, 0xa7, 0x6e, 0x75, 0x6d, 0x65, 0x72, 0x69, 0x63)
	o = msgp.AppendInt(o, z.Numeric)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *PostgresVersion) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var bzg uint32
	bzg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for bzg > 0 {
		bzg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "full":
			z.Full, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "short":
			z.Short, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "numeric":
			z.Numeric, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z PostgresVersion) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Full) + 6 + msgp.StringPrefixSize + len(z.Short) + 8 + msgp.IntSize
	return
}
