package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Explain) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "normalized_query":
			z.NormalizedQuery, err = dc.ReadString()
			if err != nil {
				return
			}
		case "runtime":
			z.Runtime, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "explain_output":
			var bai uint32
			bai, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.ExplainOutput) >= int(bai) {
				z.ExplainOutput = z.ExplainOutput[:bai]
			} else {
				z.ExplainOutput = make([]interface{}, bai)
			}
			for xvk := range z.ExplainOutput {
				z.ExplainOutput[xvk], err = dc.ReadIntf()
				if err != nil {
					return
				}
			}
		case "explain_error":
			err = z.ExplainError.DecodeMsg(dc)
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
func (z *Explain) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "occurred_at"
	err = en.Append(0x85, 0xab, 0x6f, 0x63, 0x63, 0x75, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x61, 0x74)
	if err != nil {
		return err
	}
	err = z.OccurredAt.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "normalized_query"
	err = en.Append(0xb0, 0x6e, 0x6f, 0x72, 0x6d, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteString(z.NormalizedQuery)
	if err != nil {
		return
	}
	// write "runtime"
	err = en.Append(0xa7, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Runtime)
	if err != nil {
		return
	}
	// write "explain_output"
	err = en.Append(0xae, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x69, 0x6e, 0x5f, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.ExplainOutput)))
	if err != nil {
		return
	}
	for xvk := range z.ExplainOutput {
		err = en.WriteIntf(z.ExplainOutput[xvk])
		if err != nil {
			return
		}
	}
	// write "explain_error"
	err = en.Append(0xad, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x69, 0x6e, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72)
	if err != nil {
		return err
	}
	err = z.ExplainError.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Explain) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "occurred_at"
	o = append(o, 0x85, 0xab, 0x6f, 0x63, 0x63, 0x75, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x61, 0x74)
	o, err = z.OccurredAt.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "normalized_query"
	o = append(o, 0xb0, 0x6e, 0x6f, 0x72, 0x6d, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79)
	o = msgp.AppendString(o, z.NormalizedQuery)
	// string "runtime"
	o = append(o, 0xa7, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendFloat64(o, z.Runtime)
	// string "explain_output"
	o = append(o, 0xae, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x69, 0x6e, 0x5f, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ExplainOutput)))
	for xvk := range z.ExplainOutput {
		o, err = msgp.AppendIntf(o, z.ExplainOutput[xvk])
		if err != nil {
			return
		}
	}
	// string "explain_error"
	o = append(o, 0xad, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x69, 0x6e, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72)
	o, err = z.ExplainError.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Explain) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var cmr uint32
	cmr, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for cmr > 0 {
		cmr--
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
		case "normalized_query":
			z.NormalizedQuery, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "runtime":
			z.Runtime, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "explain_output":
			var ajw uint32
			ajw, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.ExplainOutput) >= int(ajw) {
				z.ExplainOutput = z.ExplainOutput[:ajw]
			} else {
				z.ExplainOutput = make([]interface{}, ajw)
			}
			for xvk := range z.ExplainOutput {
				z.ExplainOutput[xvk], bts, err = msgp.ReadIntfBytes(bts)
				if err != nil {
					return
				}
			}
		case "explain_error":
			bts, err = z.ExplainError.UnmarshalMsg(bts)
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

func (z *Explain) Msgsize() (s int) {
	s = 1 + 12 + z.OccurredAt.Msgsize() + 17 + msgp.StringPrefixSize + len(z.NormalizedQuery) + 8 + msgp.Float64Size + 15 + msgp.ArrayHeaderSize
	for xvk := range z.ExplainOutput {
		s += msgp.GuessSize(z.ExplainOutput[xvk])
	}
	s += 14 + z.ExplainError.Msgsize()
	return
}
