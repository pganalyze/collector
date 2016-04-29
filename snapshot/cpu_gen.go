package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *CPU) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "utilization":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Utilization = nil
			} else {
				if z.Utilization == nil {
					z.Utilization = new(float64)
				}
				*z.Utilization, err = dc.ReadFloat64()
				if err != nil {
					return
				}
			}
		case "busy_times_guest_msec":
			err = z.BusyTimesGuestMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_guest_nice_msec":
			err = z.BusyTimesGuestNiceMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_idle_msec":
			err = z.BusyTimesIdleMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_iowait_msec":
			err = z.BusyTimesIowaitMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_irq_msec":
			err = z.BusyTimesIrqMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_nice_msec":
			err = z.BusyTimesNiceMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_softirq_msec":
			err = z.BusyTimesSoftirqMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_steal_msec":
			err = z.BusyTimesStealMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_system_msec":
			err = z.BusyTimesSystemMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "busy_times_user_msec":
			err = z.BusyTimesUserMsec.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "hardware_cache_size":
			err = z.HardwareCacheSize.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "hardware_model":
			err = z.HardwareModel.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "hardware_sockets":
			err = z.HardwareSockets.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "hardware_cores_per_socket":
			err = z.HardwareCoresPerSocket.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "hardware_speed_mhz":
			err = z.HardwareSpeedMhz.DecodeMsg(dc)
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
func (z *CPU) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 16
	// write "utilization"
	err = en.Append(0xde, 0x0, 0x10, 0xab, 0x75, 0x74, 0x69, 0x6c, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	if z.Utilization == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteFloat64(*z.Utilization)
		if err != nil {
			return
		}
	}
	// write "busy_times_guest_msec"
	err = en.Append(0xb5, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x67, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesGuestMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_guest_nice_msec"
	err = en.Append(0xba, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x67, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x6e, 0x69, 0x63, 0x65, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesGuestNiceMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_idle_msec"
	err = en.Append(0xb4, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x69, 0x64, 0x6c, 0x65, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesIdleMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_iowait_msec"
	err = en.Append(0xb6, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x69, 0x6f, 0x77, 0x61, 0x69, 0x74, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesIowaitMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_irq_msec"
	err = en.Append(0xb3, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x69, 0x72, 0x71, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesIrqMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_nice_msec"
	err = en.Append(0xb4, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x6e, 0x69, 0x63, 0x65, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesNiceMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_softirq_msec"
	err = en.Append(0xb7, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x73, 0x6f, 0x66, 0x74, 0x69, 0x72, 0x71, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesSoftirqMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_steal_msec"
	err = en.Append(0xb5, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x73, 0x74, 0x65, 0x61, 0x6c, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesStealMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_system_msec"
	err = en.Append(0xb6, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesSystemMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "busy_times_user_msec"
	err = en.Append(0xb4, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	if err != nil {
		return err
	}
	err = z.BusyTimesUserMsec.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "hardware_cache_size"
	err = en.Append(0xb3, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x73, 0x69, 0x7a, 0x65)
	if err != nil {
		return err
	}
	err = z.HardwareCacheSize.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "hardware_model"
	err = en.Append(0xae, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x6c)
	if err != nil {
		return err
	}
	err = z.HardwareModel.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "hardware_sockets"
	err = en.Append(0xb0, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x73, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = z.HardwareSockets.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "hardware_cores_per_socket"
	err = en.Append(0xb9, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x63, 0x6f, 0x72, 0x65, 0x73, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x73, 0x6f, 0x63, 0x6b, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = z.HardwareCoresPerSocket.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "hardware_speed_mhz"
	err = en.Append(0xb2, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x73, 0x70, 0x65, 0x65, 0x64, 0x5f, 0x6d, 0x68, 0x7a)
	if err != nil {
		return err
	}
	err = z.HardwareSpeedMhz.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *CPU) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 16
	// string "utilization"
	o = append(o, 0xde, 0x0, 0x10, 0xab, 0x75, 0x74, 0x69, 0x6c, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	if z.Utilization == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendFloat64(o, *z.Utilization)
	}
	// string "busy_times_guest_msec"
	o = append(o, 0xb5, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x67, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesGuestMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_guest_nice_msec"
	o = append(o, 0xba, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x67, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x6e, 0x69, 0x63, 0x65, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesGuestNiceMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_idle_msec"
	o = append(o, 0xb4, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x69, 0x64, 0x6c, 0x65, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesIdleMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_iowait_msec"
	o = append(o, 0xb6, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x69, 0x6f, 0x77, 0x61, 0x69, 0x74, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesIowaitMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_irq_msec"
	o = append(o, 0xb3, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x69, 0x72, 0x71, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesIrqMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_nice_msec"
	o = append(o, 0xb4, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x6e, 0x69, 0x63, 0x65, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesNiceMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_softirq_msec"
	o = append(o, 0xb7, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x73, 0x6f, 0x66, 0x74, 0x69, 0x72, 0x71, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesSoftirqMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_steal_msec"
	o = append(o, 0xb5, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x73, 0x74, 0x65, 0x61, 0x6c, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesStealMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_system_msec"
	o = append(o, 0xb6, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesSystemMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "busy_times_user_msec"
	o = append(o, 0xb4, 0x62, 0x75, 0x73, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x6d, 0x73, 0x65, 0x63)
	o, err = z.BusyTimesUserMsec.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "hardware_cache_size"
	o = append(o, 0xb3, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x73, 0x69, 0x7a, 0x65)
	o, err = z.HardwareCacheSize.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "hardware_model"
	o = append(o, 0xae, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x6c)
	o, err = z.HardwareModel.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "hardware_sockets"
	o = append(o, 0xb0, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x73, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x73)
	o, err = z.HardwareSockets.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "hardware_cores_per_socket"
	o = append(o, 0xb9, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x63, 0x6f, 0x72, 0x65, 0x73, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x73, 0x6f, 0x63, 0x6b, 0x65, 0x74)
	o, err = z.HardwareCoresPerSocket.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "hardware_speed_mhz"
	o = append(o, 0xb2, 0x68, 0x61, 0x72, 0x64, 0x77, 0x61, 0x72, 0x65, 0x5f, 0x73, 0x70, 0x65, 0x65, 0x64, 0x5f, 0x6d, 0x68, 0x7a)
	o, err = z.HardwareSpeedMhz.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *CPU) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "utilization":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Utilization = nil
			} else {
				if z.Utilization == nil {
					z.Utilization = new(float64)
				}
				*z.Utilization, bts, err = msgp.ReadFloat64Bytes(bts)
				if err != nil {
					return
				}
			}
		case "busy_times_guest_msec":
			bts, err = z.BusyTimesGuestMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_guest_nice_msec":
			bts, err = z.BusyTimesGuestNiceMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_idle_msec":
			bts, err = z.BusyTimesIdleMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_iowait_msec":
			bts, err = z.BusyTimesIowaitMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_irq_msec":
			bts, err = z.BusyTimesIrqMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_nice_msec":
			bts, err = z.BusyTimesNiceMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_softirq_msec":
			bts, err = z.BusyTimesSoftirqMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_steal_msec":
			bts, err = z.BusyTimesStealMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_system_msec":
			bts, err = z.BusyTimesSystemMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "busy_times_user_msec":
			bts, err = z.BusyTimesUserMsec.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "hardware_cache_size":
			bts, err = z.HardwareCacheSize.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "hardware_model":
			bts, err = z.HardwareModel.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "hardware_sockets":
			bts, err = z.HardwareSockets.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "hardware_cores_per_socket":
			bts, err = z.HardwareCoresPerSocket.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "hardware_speed_mhz":
			bts, err = z.HardwareSpeedMhz.UnmarshalMsg(bts)
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

func (z *CPU) Msgsize() (s int) {
	s = 3 + 12
	if z.Utilization == nil {
		s += msgp.NilSize
	} else {
		s += msgp.Float64Size
	}
	s += 22 + z.BusyTimesGuestMsec.Msgsize() + 27 + z.BusyTimesGuestNiceMsec.Msgsize() + 21 + z.BusyTimesIdleMsec.Msgsize() + 23 + z.BusyTimesIowaitMsec.Msgsize() + 20 + z.BusyTimesIrqMsec.Msgsize() + 21 + z.BusyTimesNiceMsec.Msgsize() + 24 + z.BusyTimesSoftirqMsec.Msgsize() + 22 + z.BusyTimesStealMsec.Msgsize() + 23 + z.BusyTimesSystemMsec.Msgsize() + 21 + z.BusyTimesUserMsec.Msgsize() + 20 + z.HardwareCacheSize.Msgsize() + 15 + z.HardwareModel.Msgsize() + 17 + z.HardwareSockets.Msgsize() + 26 + z.HardwareCoresPerSocket.Msgsize() + 19 + z.HardwareSpeedMhz.Msgsize()
	return
}
