package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *LogLine) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var bzg uint32
	bzg, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for bzg > 0 {
		bzg--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "occurred_at":
			err = z.OccurredAt.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "source":
			{
				var bai int
				bai, err = dc.ReadInt()
				z.Source = SourceType(bai)
			}
			if err != nil {
				return
			}
		case "client_ip":
			z.ClientIP, err = dc.ReadString()
			if err != nil {
				return
			}
		case "log_level":
			z.LogLevel, err = dc.ReadString()
			if err != nil {
				return
			}
		case "backend_pid":
			z.BackendPid, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "content":
			z.Content, err = dc.ReadString()
			if err != nil {
				return
			}
		case "additional_lines":
			var cmr uint32
			cmr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.AdditionalLines) >= int(cmr) {
				z.AdditionalLines = z.AdditionalLines[:cmr]
			} else {
				z.AdditionalLines = make([]LogLine, cmr)
			}
			for xvk := range z.AdditionalLines {
				err = z.AdditionalLines[xvk].DecodeMsg(dc)
				if err != nil {
					return
				}
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
func (z *LogLine) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "occurred_at"
	err = en.Append(0x87, 0xab, 0x6f, 0x63, 0x63, 0x75, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x61, 0x74)
	if err != nil {
		return err
	}
	err = z.OccurredAt.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "source"
	err = en.Append(0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt(int(z.Source))
	if err != nil {
		return
	}
	// write "client_ip"
	err = en.Append(0xa9, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteString(z.ClientIP)
	if err != nil {
		return
	}
	// write "log_level"
	err = en.Append(0xa9, 0x6c, 0x6f, 0x67, 0x5f, 0x6c, 0x65, 0x76, 0x65, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteString(z.LogLevel)
	if err != nil {
		return
	}
	// write "backend_pid"
	err = en.Append(0xab, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x5f, 0x70, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.BackendPid)
	if err != nil {
		return
	}
	// write "content"
	err = en.Append(0xa7, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Content)
	if err != nil {
		return
	}
	// write "additional_lines"
	err = en.Append(0xb0, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x6c, 0x69, 0x6e, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.AdditionalLines)))
	if err != nil {
		return
	}
	for xvk := range z.AdditionalLines {
		err = z.AdditionalLines[xvk].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *LogLine) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "occurred_at"
	o = append(o, 0x87, 0xab, 0x6f, 0x63, 0x63, 0x75, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x61, 0x74)
	o, err = z.OccurredAt.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "source"
	o = append(o, 0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65)
	o = msgp.AppendInt(o, int(z.Source))
	// string "client_ip"
	o = append(o, 0xa9, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x70)
	o = msgp.AppendString(o, z.ClientIP)
	// string "log_level"
	o = append(o, 0xa9, 0x6c, 0x6f, 0x67, 0x5f, 0x6c, 0x65, 0x76, 0x65, 0x6c)
	o = msgp.AppendString(o, z.LogLevel)
	// string "backend_pid"
	o = append(o, 0xab, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x5f, 0x70, 0x69, 0x64)
	o = msgp.AppendInt(o, z.BackendPid)
	// string "content"
	o = append(o, 0xa7, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74)
	o = msgp.AppendString(o, z.Content)
	// string "additional_lines"
	o = append(o, 0xb0, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x6c, 0x69, 0x6e, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.AdditionalLines)))
	for xvk := range z.AdditionalLines {
		o, err = z.AdditionalLines[xvk].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *LogLine) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var ajw uint32
	ajw, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for ajw > 0 {
		ajw--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "occurred_at":
			bts, err = z.OccurredAt.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "source":
			{
				var wht int
				wht, bts, err = msgp.ReadIntBytes(bts)
				z.Source = SourceType(wht)
			}
			if err != nil {
				return
			}
		case "client_ip":
			z.ClientIP, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "log_level":
			z.LogLevel, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "backend_pid":
			z.BackendPid, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "content":
			z.Content, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "additional_lines":
			var hct uint32
			hct, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.AdditionalLines) >= int(hct) {
				z.AdditionalLines = z.AdditionalLines[:hct]
			} else {
				z.AdditionalLines = make([]LogLine, hct)
			}
			for xvk := range z.AdditionalLines {
				bts, err = z.AdditionalLines[xvk].UnmarshalMsg(bts)
				if err != nil {
					return
				}
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

func (z *LogLine) Msgsize() (s int) {
	s = 1 + 12 + z.OccurredAt.Msgsize() + 7 + msgp.IntSize + 10 + msgp.StringPrefixSize + len(z.ClientIP) + 10 + msgp.StringPrefixSize + len(z.LogLevel) + 12 + msgp.IntSize + 8 + msgp.StringPrefixSize + len(z.Content) + 17 + msgp.ArrayHeaderSize
	for xvk := range z.AdditionalLines {
		s += z.AdditionalLines[xvk].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SourceType) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var cua int
		cua, err = dc.ReadInt()
		(*z) = SourceType(cua)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z SourceType) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteInt(int(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z SourceType) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt(o, int(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SourceType) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var xhx int
		xhx, bts, err = msgp.ReadIntBytes(bts)
		(*z) = SourceType(xhx)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

func (z SourceType) Msgsize() (s int) {
	s = msgp.IntSize
	return
}
