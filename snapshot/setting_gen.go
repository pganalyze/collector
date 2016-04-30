package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Setting) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "current_value":
			err = z.CurrentValue.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "unit":
			err = z.Unit.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "boot_value":
			err = z.BootValue.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "reset_value":
			err = z.ResetValue.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "source":
			err = z.Source.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "sourcefile":
			err = z.SourceFile.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "sourceline":
			err = z.SourceLine.DecodeMsg(dc)
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
func (z *Setting) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 8
	// write "name"
	err = en.Append(0x88, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "current_value"
	err = en.Append(0xad, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = z.CurrentValue.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "unit"
	err = en.Append(0xa4, 0x75, 0x6e, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = z.Unit.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "boot_value"
	err = en.Append(0xaa, 0x62, 0x6f, 0x6f, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = z.BootValue.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "reset_value"
	err = en.Append(0xab, 0x72, 0x65, 0x73, 0x65, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = z.ResetValue.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "source"
	err = en.Append(0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65)
	if err != nil {
		return err
	}
	err = z.Source.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "sourcefile"
	err = en.Append(0xaa, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x66, 0x69, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = z.SourceFile.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "sourceline"
	err = en.Append(0xaa, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x6c, 0x69, 0x6e, 0x65)
	if err != nil {
		return err
	}
	err = z.SourceLine.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Setting) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "name"
	o = append(o, 0x88, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "current_value"
	o = append(o, 0xad, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o, err = z.CurrentValue.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "unit"
	o = append(o, 0xa4, 0x75, 0x6e, 0x69, 0x74)
	o, err = z.Unit.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "boot_value"
	o = append(o, 0xaa, 0x62, 0x6f, 0x6f, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o, err = z.BootValue.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "reset_value"
	o = append(o, 0xab, 0x72, 0x65, 0x73, 0x65, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o, err = z.ResetValue.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "source"
	o = append(o, 0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65)
	o, err = z.Source.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "sourcefile"
	o = append(o, 0xaa, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x66, 0x69, 0x6c, 0x65)
	o, err = z.SourceFile.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "sourceline"
	o = append(o, 0xaa, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x6c, 0x69, 0x6e, 0x65)
	o, err = z.SourceLine.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Setting) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "current_value":
			bts, err = z.CurrentValue.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "unit":
			bts, err = z.Unit.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "boot_value":
			bts, err = z.BootValue.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "reset_value":
			bts, err = z.ResetValue.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "source":
			bts, err = z.Source.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "sourcefile":
			bts, err = z.SourceFile.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "sourceline":
			bts, err = z.SourceLine.UnmarshalMsg(bts)
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

func (z *Setting) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Name) + 14 + z.CurrentValue.Msgsize() + 5 + z.Unit.Msgsize() + 11 + z.BootValue.Msgsize() + 12 + z.ResetValue.Msgsize() + 7 + z.Source.Msgsize() + 11 + z.SourceFile.Msgsize() + 11 + z.SourceLine.Msgsize()
	return
}
