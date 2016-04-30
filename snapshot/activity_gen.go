package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"time"

	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Activity) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "pid":
			z.Pid, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "username":
			err = z.Username.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "application_name":
			err = z.ApplicationName.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "client_addr":
			err = z.ClientAddr.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "backend_start":
			{
				var bzg time.Time
				bzg, err = dc.ReadTime()
				z.StateChange = TimeToNullTimestamp(bzg)
			}
			if err != nil {
				return
			}
		case "xact_start":
			{
				var bai time.Time
				bai, err = dc.ReadTime()
				z.StateChange = TimeToNullTimestamp(bai)
			}
			if err != nil {
				return
			}
		case "query_start":
			{
				var cmr time.Time
				cmr, err = dc.ReadTime()
				z.StateChange = TimeToNullTimestamp(cmr)
			}
			if err != nil {
				return
			}
		case "state_change":
			{
				var ajw time.Time
				ajw, err = dc.ReadTime()
				z.StateChange = TimeToNullTimestamp(ajw)
			}
			if err != nil {
				return
			}
		case "waiting":
			err = z.Waiting.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "state":
			err = z.State.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "normalized_query":
			err = z.NormalizedQuery.DecodeMsg(dc)
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
func (z *Activity) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 11
	// write "pid"
	err = en.Append(0x8b, 0xa3, 0x70, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Pid)
	if err != nil {
		return
	}
	// write "username"
	err = en.Append(0xa8, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.Username.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "application_name"
	err = en.Append(0xb0, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.ApplicationName.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "client_addr"
	err = en.Append(0xab, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72)
	if err != nil {
		return err
	}
	err = z.ClientAddr.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "backend_start"
	err = en.Append(0xad, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteTime(NullTimestampToTime(z.StateChange))
	if err != nil {
		return
	}
	// write "xact_start"
	err = en.Append(0xaa, 0x78, 0x61, 0x63, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteTime(NullTimestampToTime(z.StateChange))
	if err != nil {
		return
	}
	// write "query_start"
	err = en.Append(0xab, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteTime(NullTimestampToTime(z.StateChange))
	if err != nil {
		return
	}
	// write "state_change"
	err = en.Append(0xac, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteTime(NullTimestampToTime(z.StateChange))
	if err != nil {
		return
	}
	// write "waiting"
	err = en.Append(0xa7, 0x77, 0x61, 0x69, 0x74, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = z.Waiting.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "state"
	err = en.Append(0xa5, 0x73, 0x74, 0x61, 0x74, 0x65)
	if err != nil {
		return err
	}
	err = z.State.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "normalized_query"
	err = en.Append(0xb0, 0x6e, 0x6f, 0x72, 0x6d, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = z.NormalizedQuery.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Activity) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 11
	// string "pid"
	o = append(o, 0x8b, 0xa3, 0x70, 0x69, 0x64)
	o = msgp.AppendInt(o, z.Pid)
	// string "username"
	o = append(o, 0xa8, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65)
	o, err = z.Username.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "application_name"
	o = append(o, 0xb0, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	o, err = z.ApplicationName.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "client_addr"
	o = append(o, 0xab, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72)
	o, err = z.ClientAddr.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "backend_start"
	o = append(o, 0xad, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendTime(o, NullTimestampToTime(z.StateChange))
	// string "xact_start"
	o = append(o, 0xaa, 0x78, 0x61, 0x63, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendTime(o, NullTimestampToTime(z.StateChange))
	// string "query_start"
	o = append(o, 0xab, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendTime(o, NullTimestampToTime(z.StateChange))
	// string "state_change"
	o = append(o, 0xac, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65)
	o = msgp.AppendTime(o, NullTimestampToTime(z.StateChange))
	// string "waiting"
	o = append(o, 0xa7, 0x77, 0x61, 0x69, 0x74, 0x69, 0x6e, 0x67)
	o, err = z.Waiting.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "state"
	o = append(o, 0xa5, 0x73, 0x74, 0x61, 0x74, 0x65)
	o, err = z.State.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "normalized_query"
	o = append(o, 0xb0, 0x6e, 0x6f, 0x72, 0x6d, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79)
	o, err = z.NormalizedQuery.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Activity) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var wht uint32
	wht, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for wht > 0 {
		wht--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "pid":
			z.Pid, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "username":
			bts, err = z.Username.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "application_name":
			bts, err = z.ApplicationName.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "client_addr":
			bts, err = z.ClientAddr.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "backend_start":
			{
				var hct time.Time
				hct, bts, err = msgp.ReadTimeBytes(bts)
				z.StateChange = TimeToNullTimestamp(hct)
			}
			if err != nil {
				return
			}
		case "xact_start":
			{
				var cua time.Time
				cua, bts, err = msgp.ReadTimeBytes(bts)
				z.StateChange = TimeToNullTimestamp(cua)
			}
			if err != nil {
				return
			}
		case "query_start":
			{
				var xhx time.Time
				xhx, bts, err = msgp.ReadTimeBytes(bts)
				z.StateChange = TimeToNullTimestamp(xhx)
			}
			if err != nil {
				return
			}
		case "state_change":
			{
				var lqf time.Time
				lqf, bts, err = msgp.ReadTimeBytes(bts)
				z.StateChange = TimeToNullTimestamp(lqf)
			}
			if err != nil {
				return
			}
		case "waiting":
			bts, err = z.Waiting.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "state":
			bts, err = z.State.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "normalized_query":
			bts, err = z.NormalizedQuery.UnmarshalMsg(bts)
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

func (z *Activity) Msgsize() (s int) {
	s = 1 + 4 + msgp.IntSize + 9 + z.Username.Msgsize() + 17 + z.ApplicationName.Msgsize() + 12 + z.ClientAddr.Msgsize() + 14 + msgp.TimeSize + 11 + msgp.TimeSize + 12 + msgp.TimeSize + 13 + msgp.TimeSize + 8 + z.Waiting.Msgsize() + 6 + z.State.Msgsize() + 17 + z.NormalizedQuery.Msgsize()
	return
}
