package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *System) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "system_type":
			{
				var bai int
				bai, err = dc.ReadInt()
				z.SystemType = SystemType(bai)
			}
			if err != nil {
				return
			}
		case "system_info":
			z.SystemInfo, err = dc.ReadIntf()
			if err != nil {
				return
			}
		case "storage":
			var cmr uint32
			cmr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Storage) >= int(cmr) {
				z.Storage = z.Storage[:cmr]
			} else {
				z.Storage = make([]Storage, cmr)
			}
			for xvk := range z.Storage {
				err = z.Storage[xvk].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "cpu":
			err = z.CPU.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "memory":
			err = z.Memory.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "network":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Network = nil
			} else {
				if z.Network == nil {
					z.Network = new(Network)
				}
				err = z.Network.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "scheduler":
			err = z.Scheduler.DecodeMsg(dc)
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
func (z *System) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "system_type"
	err = en.Append(0x87, 0xab, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x5f, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt(int(z.SystemType))
	if err != nil {
		return
	}
	// write "system_info"
	err = en.Append(0xab, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x5f, 0x69, 0x6e, 0x66, 0x6f)
	if err != nil {
		return err
	}
	err = en.WriteIntf(z.SystemInfo)
	if err != nil {
		return
	}
	// write "storage"
	err = en.Append(0xa7, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Storage)))
	if err != nil {
		return
	}
	for xvk := range z.Storage {
		err = z.Storage[xvk].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "cpu"
	err = en.Append(0xa3, 0x63, 0x70, 0x75)
	if err != nil {
		return err
	}
	err = z.CPU.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "memory"
	err = en.Append(0xa6, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = z.Memory.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "network"
	err = en.Append(0xa7, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b)
	if err != nil {
		return err
	}
	if z.Network == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Network.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "scheduler"
	err = en.Append(0xa9, 0x73, 0x63, 0x68, 0x65, 0x64, 0x75, 0x6c, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = z.Scheduler.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *System) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "system_type"
	o = append(o, 0x87, 0xab, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x5f, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendInt(o, int(z.SystemType))
	// string "system_info"
	o = append(o, 0xab, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x5f, 0x69, 0x6e, 0x66, 0x6f)
	o, err = msgp.AppendIntf(o, z.SystemInfo)
	if err != nil {
		return
	}
	// string "storage"
	o = append(o, 0xa7, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Storage)))
	for xvk := range z.Storage {
		o, err = z.Storage[xvk].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "cpu"
	o = append(o, 0xa3, 0x63, 0x70, 0x75)
	o, err = z.CPU.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "memory"
	o = append(o, 0xa6, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79)
	o, err = z.Memory.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "network"
	o = append(o, 0xa7, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b)
	if z.Network == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Network.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "scheduler"
	o = append(o, 0xa9, 0x73, 0x63, 0x68, 0x65, 0x64, 0x75, 0x6c, 0x65, 0x72)
	o, err = z.Scheduler.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *System) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "system_type":
			{
				var wht int
				wht, bts, err = msgp.ReadIntBytes(bts)
				z.SystemType = SystemType(wht)
			}
			if err != nil {
				return
			}
		case "system_info":
			z.SystemInfo, bts, err = msgp.ReadIntfBytes(bts)
			if err != nil {
				return
			}
		case "storage":
			var hct uint32
			hct, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Storage) >= int(hct) {
				z.Storage = z.Storage[:hct]
			} else {
				z.Storage = make([]Storage, hct)
			}
			for xvk := range z.Storage {
				bts, err = z.Storage[xvk].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "cpu":
			bts, err = z.CPU.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "memory":
			bts, err = z.Memory.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "network":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Network = nil
			} else {
				if z.Network == nil {
					z.Network = new(Network)
				}
				bts, err = z.Network.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "scheduler":
			bts, err = z.Scheduler.UnmarshalMsg(bts)
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

func (z *System) Msgsize() (s int) {
	s = 1 + 12 + msgp.IntSize + 12 + msgp.GuessSize(z.SystemInfo) + 8 + msgp.ArrayHeaderSize
	for xvk := range z.Storage {
		s += z.Storage[xvk].Msgsize()
	}
	s += 4 + z.CPU.Msgsize() + 7 + z.Memory.Msgsize() + 8
	if z.Network == nil {
		s += msgp.NilSize
	} else {
		s += z.Network.Msgsize()
	}
	s += 10 + z.Scheduler.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SystemType) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var cua int
		cua, err = dc.ReadInt()
		(*z) = SystemType(cua)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z SystemType) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteInt(int(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z SystemType) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt(o, int(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SystemType) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var xhx int
		xhx, bts, err = msgp.ReadIntBytes(bts)
		(*z) = SystemType(xhx)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

func (z SystemType) Msgsize() (s int) {
	s = msgp.IntSize
	return
}
