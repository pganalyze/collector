package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Statement) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "userid":
			z.Userid, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "query":
			z.Query, err = dc.ReadString()
			if err != nil {
				return
			}
		case "calls":
			z.Calls, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "total_time":
			z.TotalTime, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "rows":
			z.Rows, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "shared_blks_hit":
			z.SharedBlksHit, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "shared_blks_read":
			z.SharedBlksRead, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "shared_blks_dirtied":
			z.SharedBlksDirtied, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "shared_blks_written":
			z.SharedBlksWritten, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "local_blks_hit":
			z.LocalBlksHit, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "local_blks_read":
			z.LocalBlksRead, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "local_blks_dirtied":
			z.LocalBlksDirtied, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "local_blks_written":
			z.LocalBlksWritten, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "temp_blks_read":
			z.TempBlksRead, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "temp_blks_written":
			z.TempBlksWritten, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "blk_read_time":
			z.BlkReadTime, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "blk_write_time":
			z.BlkWriteTime, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "queryid":
			err = z.Queryid.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "min_time":
			err = z.MinTime.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "max_time":
			err = z.MaxTime.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "mean_time":
			err = z.MeanTime.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "stddev_time":
			err = z.StddevTime.DecodeMsg(dc)
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
func (z *Statement) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 22
	// write "userid"
	err = en.Append(0xde, 0x0, 0x16, 0xa6, 0x75, 0x73, 0x65, 0x72, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Userid)
	if err != nil {
		return
	}
	// write "query"
	err = en.Append(0xa5, 0x71, 0x75, 0x65, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Query)
	if err != nil {
		return
	}
	// write "calls"
	err = en.Append(0xa5, 0x63, 0x61, 0x6c, 0x6c, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Calls)
	if err != nil {
		return
	}
	// write "total_time"
	err = en.Append(0xaa, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.TotalTime)
	if err != nil {
		return
	}
	// write "rows"
	err = en.Append(0xa4, 0x72, 0x6f, 0x77, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Rows)
	if err != nil {
		return
	}
	// write "shared_blks_hit"
	err = en.Append(0xaf, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SharedBlksHit)
	if err != nil {
		return
	}
	// write "shared_blks_read"
	err = en.Append(0xb0, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SharedBlksRead)
	if err != nil {
		return
	}
	// write "shared_blks_dirtied"
	err = en.Append(0xb3, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x64, 0x69, 0x72, 0x74, 0x69, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SharedBlksDirtied)
	if err != nil {
		return
	}
	// write "shared_blks_written"
	err = en.Append(0xb3, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SharedBlksWritten)
	if err != nil {
		return
	}
	// write "local_blks_hit"
	err = en.Append(0xae, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.LocalBlksHit)
	if err != nil {
		return
	}
	// write "local_blks_read"
	err = en.Append(0xaf, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.LocalBlksRead)
	if err != nil {
		return
	}
	// write "local_blks_dirtied"
	err = en.Append(0xb2, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x64, 0x69, 0x72, 0x74, 0x69, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.LocalBlksDirtied)
	if err != nil {
		return
	}
	// write "local_blks_written"
	err = en.Append(0xb2, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.LocalBlksWritten)
	if err != nil {
		return
	}
	// write "temp_blks_read"
	err = en.Append(0xae, 0x74, 0x65, 0x6d, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.TempBlksRead)
	if err != nil {
		return
	}
	// write "temp_blks_written"
	err = en.Append(0xb1, 0x74, 0x65, 0x6d, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.TempBlksWritten)
	if err != nil {
		return
	}
	// write "blk_read_time"
	err = en.Append(0xad, 0x62, 0x6c, 0x6b, 0x5f, 0x72, 0x65, 0x61, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.BlkReadTime)
	if err != nil {
		return
	}
	// write "blk_write_time"
	err = en.Append(0xae, 0x62, 0x6c, 0x6b, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.BlkWriteTime)
	if err != nil {
		return
	}
	// write "queryid"
	err = en.Append(0xa7, 0x71, 0x75, 0x65, 0x72, 0x79, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = z.Queryid.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "min_time"
	err = en.Append(0xa8, 0x6d, 0x69, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.MinTime.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "max_time"
	err = en.Append(0xa8, 0x6d, 0x61, 0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.MaxTime.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "mean_time"
	err = en.Append(0xa9, 0x6d, 0x65, 0x61, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.MeanTime.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "stddev_time"
	err = en.Append(0xab, 0x73, 0x74, 0x64, 0x64, 0x65, 0x76, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.StddevTime.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Statement) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 22
	// string "userid"
	o = append(o, 0xde, 0x0, 0x16, 0xa6, 0x75, 0x73, 0x65, 0x72, 0x69, 0x64)
	o = msgp.AppendInt(o, z.Userid)
	// string "query"
	o = append(o, 0xa5, 0x71, 0x75, 0x65, 0x72, 0x79)
	o = msgp.AppendString(o, z.Query)
	// string "calls"
	o = append(o, 0xa5, 0x63, 0x61, 0x6c, 0x6c, 0x73)
	o = msgp.AppendInt64(o, z.Calls)
	// string "total_time"
	o = append(o, 0xaa, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendFloat64(o, z.TotalTime)
	// string "rows"
	o = append(o, 0xa4, 0x72, 0x6f, 0x77, 0x73)
	o = msgp.AppendInt64(o, z.Rows)
	// string "shared_blks_hit"
	o = append(o, 0xaf, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o = msgp.AppendInt64(o, z.SharedBlksHit)
	// string "shared_blks_read"
	o = append(o, 0xb0, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o = msgp.AppendInt64(o, z.SharedBlksRead)
	// string "shared_blks_dirtied"
	o = append(o, 0xb3, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x64, 0x69, 0x72, 0x74, 0x69, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.SharedBlksDirtied)
	// string "shared_blks_written"
	o = append(o, 0xb3, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	o = msgp.AppendInt64(o, z.SharedBlksWritten)
	// string "local_blks_hit"
	o = append(o, 0xae, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o = msgp.AppendInt64(o, z.LocalBlksHit)
	// string "local_blks_read"
	o = append(o, 0xaf, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o = msgp.AppendInt64(o, z.LocalBlksRead)
	// string "local_blks_dirtied"
	o = append(o, 0xb2, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x64, 0x69, 0x72, 0x74, 0x69, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.LocalBlksDirtied)
	// string "local_blks_written"
	o = append(o, 0xb2, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	o = msgp.AppendInt64(o, z.LocalBlksWritten)
	// string "temp_blks_read"
	o = append(o, 0xae, 0x74, 0x65, 0x6d, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o = msgp.AppendInt64(o, z.TempBlksRead)
	// string "temp_blks_written"
	o = append(o, 0xb1, 0x74, 0x65, 0x6d, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	o = msgp.AppendInt64(o, z.TempBlksWritten)
	// string "blk_read_time"
	o = append(o, 0xad, 0x62, 0x6c, 0x6b, 0x5f, 0x72, 0x65, 0x61, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendFloat64(o, z.BlkReadTime)
	// string "blk_write_time"
	o = append(o, 0xae, 0x62, 0x6c, 0x6b, 0x5f, 0x77, 0x72, 0x69, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendFloat64(o, z.BlkWriteTime)
	// string "queryid"
	o = append(o, 0xa7, 0x71, 0x75, 0x65, 0x72, 0x79, 0x69, 0x64)
	o, err = z.Queryid.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "min_time"
	o = append(o, 0xa8, 0x6d, 0x69, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.MinTime.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "max_time"
	o = append(o, 0xa8, 0x6d, 0x61, 0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.MaxTime.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "mean_time"
	o = append(o, 0xa9, 0x6d, 0x65, 0x61, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.MeanTime.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "stddev_time"
	o = append(o, 0xab, 0x73, 0x74, 0x64, 0x64, 0x65, 0x76, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.StddevTime.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Statement) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "userid":
			z.Userid, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "query":
			z.Query, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "calls":
			z.Calls, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "total_time":
			z.TotalTime, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "rows":
			z.Rows, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "shared_blks_hit":
			z.SharedBlksHit, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "shared_blks_read":
			z.SharedBlksRead, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "shared_blks_dirtied":
			z.SharedBlksDirtied, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "shared_blks_written":
			z.SharedBlksWritten, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "local_blks_hit":
			z.LocalBlksHit, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "local_blks_read":
			z.LocalBlksRead, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "local_blks_dirtied":
			z.LocalBlksDirtied, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "local_blks_written":
			z.LocalBlksWritten, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "temp_blks_read":
			z.TempBlksRead, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "temp_blks_written":
			z.TempBlksWritten, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "blk_read_time":
			z.BlkReadTime, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "blk_write_time":
			z.BlkWriteTime, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "queryid":
			bts, err = z.Queryid.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "min_time":
			bts, err = z.MinTime.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "max_time":
			bts, err = z.MaxTime.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "mean_time":
			bts, err = z.MeanTime.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "stddev_time":
			bts, err = z.StddevTime.UnmarshalMsg(bts)
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

func (z *Statement) Msgsize() (s int) {
	s = 3 + 7 + msgp.IntSize + 6 + msgp.StringPrefixSize + len(z.Query) + 6 + msgp.Int64Size + 11 + msgp.Float64Size + 5 + msgp.Int64Size + 16 + msgp.Int64Size + 17 + msgp.Int64Size + 20 + msgp.Int64Size + 20 + msgp.Int64Size + 15 + msgp.Int64Size + 16 + msgp.Int64Size + 19 + msgp.Int64Size + 19 + msgp.Int64Size + 15 + msgp.Int64Size + 18 + msgp.Int64Size + 14 + msgp.Float64Size + 15 + msgp.Float64Size + 8 + z.Queryid.Msgsize() + 9 + z.MinTime.Msgsize() + 9 + z.MaxTime.Msgsize() + 10 + z.MeanTime.Msgsize() + 12 + z.StddevTime.Msgsize()
	return
}
