package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Network) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "receive_throughput":
			err = z.ReceiveThroughput.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "transmit_throughput":
			err = z.TransmitThroughput.DecodeMsg(dc)
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
func (z *Network) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "receive_throughput"
	err = en.Append(0x82, 0xb2, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	if err != nil {
		return err
	}
	err = z.ReceiveThroughput.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "transmit_throughput"
	err = en.Append(0xb3, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	if err != nil {
		return err
	}
	err = z.TransmitThroughput.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Network) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "receive_throughput"
	o = append(o, 0x82, 0xb2, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	o, err = z.ReceiveThroughput.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "transmit_throughput"
	o = append(o, 0xb3, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x74, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	o, err = z.TransmitThroughput.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Network) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "receive_throughput":
			bts, err = z.ReceiveThroughput.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "transmit_throughput":
			bts, err = z.TransmitThroughput.UnmarshalMsg(bts)
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

func (z *Network) Msgsize() (s int) {
	s = 1 + 19 + z.ReceiveThroughput.Msgsize() + 20 + z.TransmitThroughput.Msgsize()
	return
}
