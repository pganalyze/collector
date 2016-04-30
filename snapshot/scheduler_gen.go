package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Scheduler) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "context_switches":
			err = z.ContextSwitches.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "interrupts":
			err = z.Interrupts.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "loadavg_1min":
			err = z.Loadavg1min.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "loadavg_5min":
			err = z.Loadavg5min.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "loadavg_15min":
			err = z.Loadavg15min.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "procs_blocked":
			err = z.ProcsBlocked.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "procs_created":
			err = z.ProcsCreated.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "procs_running":
			err = z.ProcsRunning.DecodeMsg(dc)
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
func (z *Scheduler) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 8
	// write "context_switches"
	err = en.Append(0x88, 0xb0, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x5f, 0x73, 0x77, 0x69, 0x74, 0x63, 0x68, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.ContextSwitches.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "interrupts"
	err = en.Append(0xaa, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x72, 0x75, 0x70, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = z.Interrupts.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "loadavg_1min"
	err = en.Append(0xac, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x76, 0x67, 0x5f, 0x31, 0x6d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = z.Loadavg1min.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "loadavg_5min"
	err = en.Append(0xac, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x76, 0x67, 0x5f, 0x35, 0x6d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = z.Loadavg5min.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "loadavg_15min"
	err = en.Append(0xad, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x76, 0x67, 0x5f, 0x31, 0x35, 0x6d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = z.Loadavg15min.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "procs_blocked"
	err = en.Append(0xad, 0x70, 0x72, 0x6f, 0x63, 0x73, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = z.ProcsBlocked.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "procs_created"
	err = en.Append(0xad, 0x70, 0x72, 0x6f, 0x63, 0x73, 0x5f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = z.ProcsCreated.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "procs_running"
	err = en.Append(0xad, 0x70, 0x72, 0x6f, 0x63, 0x73, 0x5f, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = z.ProcsRunning.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Scheduler) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "context_switches"
	o = append(o, 0x88, 0xb0, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x5f, 0x73, 0x77, 0x69, 0x74, 0x63, 0x68, 0x65, 0x73)
	o, err = z.ContextSwitches.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "interrupts"
	o = append(o, 0xaa, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x72, 0x75, 0x70, 0x74, 0x73)
	o, err = z.Interrupts.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "loadavg_1min"
	o = append(o, 0xac, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x76, 0x67, 0x5f, 0x31, 0x6d, 0x69, 0x6e)
	o, err = z.Loadavg1min.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "loadavg_5min"
	o = append(o, 0xac, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x76, 0x67, 0x5f, 0x35, 0x6d, 0x69, 0x6e)
	o, err = z.Loadavg5min.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "loadavg_15min"
	o = append(o, 0xad, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x76, 0x67, 0x5f, 0x31, 0x35, 0x6d, 0x69, 0x6e)
	o, err = z.Loadavg15min.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "procs_blocked"
	o = append(o, 0xad, 0x70, 0x72, 0x6f, 0x63, 0x73, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64)
	o, err = z.ProcsBlocked.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "procs_created"
	o = append(o, 0xad, 0x70, 0x72, 0x6f, 0x63, 0x73, 0x5f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64)
	o, err = z.ProcsCreated.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "procs_running"
	o = append(o, 0xad, 0x70, 0x72, 0x6f, 0x63, 0x73, 0x5f, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67)
	o, err = z.ProcsRunning.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Scheduler) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "context_switches":
			bts, err = z.ContextSwitches.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "interrupts":
			bts, err = z.Interrupts.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "loadavg_1min":
			bts, err = z.Loadavg1min.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "loadavg_5min":
			bts, err = z.Loadavg5min.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "loadavg_15min":
			bts, err = z.Loadavg15min.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "procs_blocked":
			bts, err = z.ProcsBlocked.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "procs_created":
			bts, err = z.ProcsCreated.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "procs_running":
			bts, err = z.ProcsRunning.UnmarshalMsg(bts)
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

func (z *Scheduler) Msgsize() (s int) {
	s = 1 + 17 + z.ContextSwitches.Msgsize() + 11 + z.Interrupts.Msgsize() + 13 + z.Loadavg1min.Msgsize() + 13 + z.Loadavg5min.Msgsize() + 14 + z.Loadavg15min.Msgsize() + 14 + z.ProcsBlocked.Msgsize() + 14 + z.ProcsCreated.Msgsize() + 14 + z.ProcsRunning.Msgsize()
	return
}
