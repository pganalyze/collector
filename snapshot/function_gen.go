package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Function) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "schema_name":
			z.SchemaName, err = dc.ReadString()
			if err != nil {
				return
			}
		case "function_name":
			z.FunctionName, err = dc.ReadString()
			if err != nil {
				return
			}
		case "language":
			z.Language, err = dc.ReadString()
			if err != nil {
				return
			}
		case "source":
			z.Source, err = dc.ReadString()
			if err != nil {
				return
			}
		case "source_bin":
			err = z.SourceBin.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "config":
			err = z.Config.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "arguments":
			err = z.Arguments.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "result":
			err = z.Result.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "aggregate":
			z.Aggregate, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "window":
			z.Window, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "security_definer":
			z.SecurityDefiner, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "leakproof":
			z.Leakproof, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "strict":
			z.Strict, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "returns_set":
			z.ReturnsSet, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "volatile":
			err = z.Volatile.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "calls":
			err = z.Calls.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "total_time":
			err = z.TotalTime.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "self_time":
			err = z.SelfTime.DecodeMsg(dc)
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
func (z *Function) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 18
	// write "schema_name"
	err = en.Append(0xde, 0x0, 0x12, 0xab, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.SchemaName)
	if err != nil {
		return
	}
	// write "function_name"
	err = en.Append(0xad, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.FunctionName)
	if err != nil {
		return
	}
	// write "language"
	err = en.Append(0xa8, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Language)
	if err != nil {
		return
	}
	// write "source"
	err = en.Append(0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Source)
	if err != nil {
		return
	}
	// write "source_bin"
	err = en.Append(0xaa, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x62, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = z.SourceBin.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "config"
	err = en.Append(0xa6, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67)
	if err != nil {
		return err
	}
	err = z.Config.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "arguments"
	err = en.Append(0xa9, 0x61, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = z.Arguments.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "result"
	err = en.Append(0xa6, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74)
	if err != nil {
		return err
	}
	err = z.Result.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "aggregate"
	err = en.Append(0xa9, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Aggregate)
	if err != nil {
		return
	}
	// write "window"
	err = en.Append(0xa6, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Window)
	if err != nil {
		return
	}
	// write "security_definer"
	err = en.Append(0xb0, 0x73, 0x65, 0x63, 0x75, 0x72, 0x69, 0x74, 0x79, 0x5f, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.SecurityDefiner)
	if err != nil {
		return
	}
	// write "leakproof"
	err = en.Append(0xa9, 0x6c, 0x65, 0x61, 0x6b, 0x70, 0x72, 0x6f, 0x6f, 0x66)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Leakproof)
	if err != nil {
		return
	}
	// write "strict"
	err = en.Append(0xa6, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Strict)
	if err != nil {
		return
	}
	// write "returns_set"
	err = en.Append(0xab, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x73, 0x5f, 0x73, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.ReturnsSet)
	if err != nil {
		return
	}
	// write "volatile"
	err = en.Append(0xa8, 0x76, 0x6f, 0x6c, 0x61, 0x74, 0x69, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = z.Volatile.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "calls"
	err = en.Append(0xa5, 0x63, 0x61, 0x6c, 0x6c, 0x73)
	if err != nil {
		return err
	}
	err = z.Calls.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "total_time"
	err = en.Append(0xaa, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.TotalTime.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "self_time"
	err = en.Append(0xa9, 0x73, 0x65, 0x6c, 0x66, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.SelfTime.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Function) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 18
	// string "schema_name"
	o = append(o, 0xde, 0x0, 0x12, 0xab, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.SchemaName)
	// string "function_name"
	o = append(o, 0xad, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.FunctionName)
	// string "language"
	o = append(o, 0xa8, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65)
	o = msgp.AppendString(o, z.Language)
	// string "source"
	o = append(o, 0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65)
	o = msgp.AppendString(o, z.Source)
	// string "source_bin"
	o = append(o, 0xaa, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x62, 0x69, 0x6e)
	o, err = z.SourceBin.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "config"
	o = append(o, 0xa6, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67)
	o, err = z.Config.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "arguments"
	o = append(o, 0xa9, 0x61, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73)
	o, err = z.Arguments.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "result"
	o = append(o, 0xa6, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74)
	o, err = z.Result.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "aggregate"
	o = append(o, 0xa9, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65)
	o = msgp.AppendBool(o, z.Aggregate)
	// string "window"
	o = append(o, 0xa6, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77)
	o = msgp.AppendBool(o, z.Window)
	// string "security_definer"
	o = append(o, 0xb0, 0x73, 0x65, 0x63, 0x75, 0x72, 0x69, 0x74, 0x79, 0x5f, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x65, 0x72)
	o = msgp.AppendBool(o, z.SecurityDefiner)
	// string "leakproof"
	o = append(o, 0xa9, 0x6c, 0x65, 0x61, 0x6b, 0x70, 0x72, 0x6f, 0x6f, 0x66)
	o = msgp.AppendBool(o, z.Leakproof)
	// string "strict"
	o = append(o, 0xa6, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74)
	o = msgp.AppendBool(o, z.Strict)
	// string "returns_set"
	o = append(o, 0xab, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x73, 0x5f, 0x73, 0x65, 0x74)
	o = msgp.AppendBool(o, z.ReturnsSet)
	// string "volatile"
	o = append(o, 0xa8, 0x76, 0x6f, 0x6c, 0x61, 0x74, 0x69, 0x6c, 0x65)
	o, err = z.Volatile.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "calls"
	o = append(o, 0xa5, 0x63, 0x61, 0x6c, 0x6c, 0x73)
	o, err = z.Calls.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "total_time"
	o = append(o, 0xaa, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.TotalTime.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "self_time"
	o = append(o, 0xa9, 0x73, 0x65, 0x6c, 0x66, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.SelfTime.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Function) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "schema_name":
			z.SchemaName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "function_name":
			z.FunctionName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "language":
			z.Language, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "source":
			z.Source, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "source_bin":
			bts, err = z.SourceBin.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "config":
			bts, err = z.Config.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "arguments":
			bts, err = z.Arguments.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "result":
			bts, err = z.Result.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "aggregate":
			z.Aggregate, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "window":
			z.Window, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "security_definer":
			z.SecurityDefiner, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "leakproof":
			z.Leakproof, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "strict":
			z.Strict, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "returns_set":
			z.ReturnsSet, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "volatile":
			bts, err = z.Volatile.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "calls":
			bts, err = z.Calls.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "total_time":
			bts, err = z.TotalTime.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "self_time":
			bts, err = z.SelfTime.UnmarshalMsg(bts)
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

func (z *Function) Msgsize() (s int) {
	s = 3 + 12 + msgp.StringPrefixSize + len(z.SchemaName) + 14 + msgp.StringPrefixSize + len(z.FunctionName) + 9 + msgp.StringPrefixSize + len(z.Language) + 7 + msgp.StringPrefixSize + len(z.Source) + 11 + z.SourceBin.Msgsize() + 7 + z.Config.Msgsize() + 10 + z.Arguments.Msgsize() + 7 + z.Result.Msgsize() + 10 + msgp.BoolSize + 7 + msgp.BoolSize + 17 + msgp.BoolSize + 10 + msgp.BoolSize + 7 + msgp.BoolSize + 12 + msgp.BoolSize + 9 + z.Volatile.Msgsize() + 6 + z.Calls.Msgsize() + 11 + z.TotalTime.Msgsize() + 10 + z.SelfTime.Msgsize()
	return
}
