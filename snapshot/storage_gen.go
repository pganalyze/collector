package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Storage) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "bytes_available":
			err = z.BytesAvailable.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "bytes_total":
			err = z.BytesTotal.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "mountpoint":
			err = z.Mountpoint.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "name":
			err = z.Name.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "path":
			err = z.Path.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "perfdata":
			err = z.Perfdata.DecodeMsg(dc)
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
func (z *Storage) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "bytes_available"
	err = en.Append(0x86, 0xaf, 0x62, 0x79, 0x74, 0x65, 0x73, 0x5f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = z.BytesAvailable.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "bytes_total"
	err = en.Append(0xab, 0x62, 0x79, 0x74, 0x65, 0x73, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = z.BytesTotal.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "mountpoint"
	err = en.Append(0xaa, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x70, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = z.Mountpoint.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "name"
	err = en.Append(0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.Name.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "path"
	err = en.Append(0xa4, 0x70, 0x61, 0x74, 0x68)
	if err != nil {
		return err
	}
	err = z.Path.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "perfdata"
	err = en.Append(0xa8, 0x70, 0x65, 0x72, 0x66, 0x64, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = z.Perfdata.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Storage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "bytes_available"
	o = append(o, 0x86, 0xaf, 0x62, 0x79, 0x74, 0x65, 0x73, 0x5f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65)
	o, err = z.BytesAvailable.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "bytes_total"
	o = append(o, 0xab, 0x62, 0x79, 0x74, 0x65, 0x73, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o, err = z.BytesTotal.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "mountpoint"
	o = append(o, 0xaa, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x70, 0x6f, 0x69, 0x6e, 0x74)
	o, err = z.Mountpoint.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "name"
	o = append(o, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o, err = z.Name.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "path"
	o = append(o, 0xa4, 0x70, 0x61, 0x74, 0x68)
	o, err = z.Path.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "perfdata"
	o = append(o, 0xa8, 0x70, 0x65, 0x72, 0x66, 0x64, 0x61, 0x74, 0x61)
	o, err = z.Perfdata.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Storage) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "bytes_available":
			bts, err = z.BytesAvailable.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "bytes_total":
			bts, err = z.BytesTotal.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "mountpoint":
			bts, err = z.Mountpoint.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "name":
			bts, err = z.Name.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "path":
			bts, err = z.Path.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "perfdata":
			bts, err = z.Perfdata.UnmarshalMsg(bts)
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

func (z *Storage) Msgsize() (s int) {
	s = 1 + 16 + z.BytesAvailable.Msgsize() + 12 + z.BytesTotal.Msgsize() + 11 + z.Mountpoint.Msgsize() + 5 + z.Name.Msgsize() + 5 + z.Path.Msgsize() + 9 + z.Perfdata.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StoragePerfdata) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var bai uint32
	bai, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for bai > 0 {
		bai--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "version":
			z.Version, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "rd_ios":
			err = z.ReadIops.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "wr_ios":
			err = z.WriteIops.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "ios_in_prog":
			err = z.IopsInProgress.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "avg_req_size":
			err = z.AvgReqSize.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "rd_latency":
			err = z.ReadLatency.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "rd_throughput":
			err = z.ReadThroughput.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "wr_latency":
			err = z.WriteLatency.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "wr_throughput":
			err = z.WriteThroughput.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "rd_merges":
			err = z.ReadMerges.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "rd_sectors":
			err = z.ReadSectors.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "rd_ticks":
			err = z.ReadTicks.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "wr_merges":
			err = z.WriteMerges.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "wr_sectors":
			err = z.WriteSectors.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "wr_ticks":
			err = z.WriteTicks.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "tot_ticks":
			err = z.TotalTicks.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "rq_ticks":
			err = z.RequestTicks.DecodeMsg(dc)
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
func (z *StoragePerfdata) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 17
	// write "version"
	err = en.Append(0xde, 0x0, 0x11, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Version)
	if err != nil {
		return
	}
	// write "rd_ios"
	err = en.Append(0xa6, 0x72, 0x64, 0x5f, 0x69, 0x6f, 0x73)
	if err != nil {
		return err
	}
	err = z.ReadIops.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "wr_ios"
	err = en.Append(0xa6, 0x77, 0x72, 0x5f, 0x69, 0x6f, 0x73)
	if err != nil {
		return err
	}
	err = z.WriteIops.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "ios_in_prog"
	err = en.Append(0xab, 0x69, 0x6f, 0x73, 0x5f, 0x69, 0x6e, 0x5f, 0x70, 0x72, 0x6f, 0x67)
	if err != nil {
		return err
	}
	err = z.IopsInProgress.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "avg_req_size"
	err = en.Append(0xac, 0x61, 0x76, 0x67, 0x5f, 0x72, 0x65, 0x71, 0x5f, 0x73, 0x69, 0x7a, 0x65)
	if err != nil {
		return err
	}
	err = z.AvgReqSize.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "rd_latency"
	err = en.Append(0xaa, 0x72, 0x64, 0x5f, 0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79)
	if err != nil {
		return err
	}
	err = z.ReadLatency.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "rd_throughput"
	err = en.Append(0xad, 0x72, 0x64, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	if err != nil {
		return err
	}
	err = z.ReadThroughput.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "wr_latency"
	err = en.Append(0xaa, 0x77, 0x72, 0x5f, 0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79)
	if err != nil {
		return err
	}
	err = z.WriteLatency.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "wr_throughput"
	err = en.Append(0xad, 0x77, 0x72, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	if err != nil {
		return err
	}
	err = z.WriteThroughput.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "rd_merges"
	err = en.Append(0xa9, 0x72, 0x64, 0x5f, 0x6d, 0x65, 0x72, 0x67, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.ReadMerges.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "rd_sectors"
	err = en.Append(0xaa, 0x72, 0x64, 0x5f, 0x73, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73)
	if err != nil {
		return err
	}
	err = z.ReadSectors.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "rd_ticks"
	err = en.Append(0xa8, 0x72, 0x64, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	if err != nil {
		return err
	}
	err = z.ReadTicks.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "wr_merges"
	err = en.Append(0xa9, 0x77, 0x72, 0x5f, 0x6d, 0x65, 0x72, 0x67, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.WriteMerges.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "wr_sectors"
	err = en.Append(0xaa, 0x77, 0x72, 0x5f, 0x73, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73)
	if err != nil {
		return err
	}
	err = z.WriteSectors.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "wr_ticks"
	err = en.Append(0xa8, 0x77, 0x72, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	if err != nil {
		return err
	}
	err = z.WriteTicks.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "tot_ticks"
	err = en.Append(0xa9, 0x74, 0x6f, 0x74, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	if err != nil {
		return err
	}
	err = z.TotalTicks.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "rq_ticks"
	err = en.Append(0xa8, 0x72, 0x71, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	if err != nil {
		return err
	}
	err = z.RequestTicks.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *StoragePerfdata) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 17
	// string "version"
	o = append(o, 0xde, 0x0, 0x11, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt(o, z.Version)
	// string "rd_ios"
	o = append(o, 0xa6, 0x72, 0x64, 0x5f, 0x69, 0x6f, 0x73)
	o, err = z.ReadIops.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "wr_ios"
	o = append(o, 0xa6, 0x77, 0x72, 0x5f, 0x69, 0x6f, 0x73)
	o, err = z.WriteIops.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "ios_in_prog"
	o = append(o, 0xab, 0x69, 0x6f, 0x73, 0x5f, 0x69, 0x6e, 0x5f, 0x70, 0x72, 0x6f, 0x67)
	o, err = z.IopsInProgress.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "avg_req_size"
	o = append(o, 0xac, 0x61, 0x76, 0x67, 0x5f, 0x72, 0x65, 0x71, 0x5f, 0x73, 0x69, 0x7a, 0x65)
	o, err = z.AvgReqSize.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "rd_latency"
	o = append(o, 0xaa, 0x72, 0x64, 0x5f, 0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79)
	o, err = z.ReadLatency.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "rd_throughput"
	o = append(o, 0xad, 0x72, 0x64, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	o, err = z.ReadThroughput.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "wr_latency"
	o = append(o, 0xaa, 0x77, 0x72, 0x5f, 0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79)
	o, err = z.WriteLatency.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "wr_throughput"
	o = append(o, 0xad, 0x77, 0x72, 0x5f, 0x74, 0x68, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x70, 0x75, 0x74)
	o, err = z.WriteThroughput.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "rd_merges"
	o = append(o, 0xa9, 0x72, 0x64, 0x5f, 0x6d, 0x65, 0x72, 0x67, 0x65, 0x73)
	o, err = z.ReadMerges.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "rd_sectors"
	o = append(o, 0xaa, 0x72, 0x64, 0x5f, 0x73, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73)
	o, err = z.ReadSectors.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "rd_ticks"
	o = append(o, 0xa8, 0x72, 0x64, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	o, err = z.ReadTicks.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "wr_merges"
	o = append(o, 0xa9, 0x77, 0x72, 0x5f, 0x6d, 0x65, 0x72, 0x67, 0x65, 0x73)
	o, err = z.WriteMerges.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "wr_sectors"
	o = append(o, 0xaa, 0x77, 0x72, 0x5f, 0x73, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73)
	o, err = z.WriteSectors.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "wr_ticks"
	o = append(o, 0xa8, 0x77, 0x72, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	o, err = z.WriteTicks.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "tot_ticks"
	o = append(o, 0xa9, 0x74, 0x6f, 0x74, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	o, err = z.TotalTicks.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "rq_ticks"
	o = append(o, 0xa8, 0x72, 0x71, 0x5f, 0x74, 0x69, 0x63, 0x6b, 0x73)
	o, err = z.RequestTicks.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StoragePerfdata) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "version":
			z.Version, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "rd_ios":
			bts, err = z.ReadIops.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "wr_ios":
			bts, err = z.WriteIops.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "ios_in_prog":
			bts, err = z.IopsInProgress.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "avg_req_size":
			bts, err = z.AvgReqSize.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "rd_latency":
			bts, err = z.ReadLatency.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "rd_throughput":
			bts, err = z.ReadThroughput.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "wr_latency":
			bts, err = z.WriteLatency.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "wr_throughput":
			bts, err = z.WriteThroughput.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "rd_merges":
			bts, err = z.ReadMerges.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "rd_sectors":
			bts, err = z.ReadSectors.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "rd_ticks":
			bts, err = z.ReadTicks.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "wr_merges":
			bts, err = z.WriteMerges.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "wr_sectors":
			bts, err = z.WriteSectors.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "wr_ticks":
			bts, err = z.WriteTicks.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "tot_ticks":
			bts, err = z.TotalTicks.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "rq_ticks":
			bts, err = z.RequestTicks.UnmarshalMsg(bts)
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

func (z *StoragePerfdata) Msgsize() (s int) {
	s = 3 + 8 + msgp.IntSize + 7 + z.ReadIops.Msgsize() + 7 + z.WriteIops.Msgsize() + 12 + z.IopsInProgress.Msgsize() + 13 + z.AvgReqSize.Msgsize() + 11 + z.ReadLatency.Msgsize() + 14 + z.ReadThroughput.Msgsize() + 11 + z.WriteLatency.Msgsize() + 14 + z.WriteThroughput.Msgsize() + 10 + z.ReadMerges.Msgsize() + 11 + z.ReadSectors.Msgsize() + 9 + z.ReadTicks.Msgsize() + 10 + z.WriteMerges.Msgsize() + 11 + z.WriteSectors.Msgsize() + 9 + z.WriteTicks.Msgsize() + 10 + z.TotalTicks.Msgsize() + 9 + z.RequestTicks.Msgsize()
	return
}
