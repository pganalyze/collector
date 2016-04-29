package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Memory) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "applications_bytes":
			err = z.ApplicationsBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "buffers_bytes":
			err = z.BuffersBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "dirty_bytes":
			err = z.DirtyBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "free_bytes":
			err = z.FreeBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "pagecache_bytes":
			err = z.PagecacheBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "swap_free_bytes":
			err = z.SwapFreeBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "swap_total_bytes":
			err = z.SwapTotalBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "total_bytes":
			err = z.TotalBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "writeback_bytes":
			err = z.WritebackBytes.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "active_bytes":
			err = z.ActiveBytes.DecodeMsg(dc)
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
func (z *Memory) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 10
	// write "applications_bytes"
	err = en.Append(0x8a, 0xb2, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.ApplicationsBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "buffers_bytes"
	err = en.Append(0xad, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72, 0x73, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.BuffersBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "dirty_bytes"
	err = en.Append(0xab, 0x64, 0x69, 0x72, 0x74, 0x79, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.DirtyBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "free_bytes"
	err = en.Append(0xaa, 0x66, 0x72, 0x65, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.FreeBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "pagecache_bytes"
	err = en.Append(0xaf, 0x70, 0x61, 0x67, 0x65, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.PagecacheBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "swap_free_bytes"
	err = en.Append(0xaf, 0x73, 0x77, 0x61, 0x70, 0x5f, 0x66, 0x72, 0x65, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.SwapFreeBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "swap_total_bytes"
	err = en.Append(0xb0, 0x73, 0x77, 0x61, 0x70, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.SwapTotalBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "total_bytes"
	err = en.Append(0xab, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.TotalBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "writeback_bytes"
	err = en.Append(0xaf, 0x77, 0x72, 0x69, 0x74, 0x65, 0x62, 0x61, 0x63, 0x6b, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.WritebackBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "active_bytes"
	err = en.Append(0xac, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.ActiveBytes.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Memory) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "applications_bytes"
	o = append(o, 0x8a, 0xb2, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.ApplicationsBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "buffers_bytes"
	o = append(o, 0xad, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72, 0x73, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.BuffersBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "dirty_bytes"
	o = append(o, 0xab, 0x64, 0x69, 0x72, 0x74, 0x79, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.DirtyBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "free_bytes"
	o = append(o, 0xaa, 0x66, 0x72, 0x65, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.FreeBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "pagecache_bytes"
	o = append(o, 0xaf, 0x70, 0x61, 0x67, 0x65, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.PagecacheBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "swap_free_bytes"
	o = append(o, 0xaf, 0x73, 0x77, 0x61, 0x70, 0x5f, 0x66, 0x72, 0x65, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.SwapFreeBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "swap_total_bytes"
	o = append(o, 0xb0, 0x73, 0x77, 0x61, 0x70, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.SwapTotalBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "total_bytes"
	o = append(o, 0xab, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.TotalBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "writeback_bytes"
	o = append(o, 0xaf, 0x77, 0x72, 0x69, 0x74, 0x65, 0x62, 0x61, 0x63, 0x6b, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.WritebackBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "active_bytes"
	o = append(o, 0xac, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o, err = z.ActiveBytes.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Memory) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "applications_bytes":
			bts, err = z.ApplicationsBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "buffers_bytes":
			bts, err = z.BuffersBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "dirty_bytes":
			bts, err = z.DirtyBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "free_bytes":
			bts, err = z.FreeBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "pagecache_bytes":
			bts, err = z.PagecacheBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "swap_free_bytes":
			bts, err = z.SwapFreeBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "swap_total_bytes":
			bts, err = z.SwapTotalBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "total_bytes":
			bts, err = z.TotalBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "writeback_bytes":
			bts, err = z.WritebackBytes.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "active_bytes":
			bts, err = z.ActiveBytes.UnmarshalMsg(bts)
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

func (z *Memory) Msgsize() (s int) {
	s = 1 + 19 + z.ApplicationsBytes.Msgsize() + 14 + z.BuffersBytes.Msgsize() + 12 + z.DirtyBytes.Msgsize() + 11 + z.FreeBytes.Msgsize() + 16 + z.PagecacheBytes.Msgsize() + 16 + z.SwapFreeBytes.Msgsize() + 17 + z.SwapTotalBytes.Msgsize() + 12 + z.TotalBytes.Msgsize() + 16 + z.WritebackBytes.Msgsize() + 13 + z.ActiveBytes.Msgsize()
	return
}
