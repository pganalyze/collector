package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Column) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "data_type":
			z.DataType, err = dc.ReadString()
			if err != nil {
				return
			}
		case "default_value":
			err = z.DefaultValue.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "not_null":
			z.NotNull, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "position":
			z.Position, err = dc.ReadInt32()
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
func (z *Column) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "name"
	err = en.Append(0x85, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "data_type"
	err = en.Append(0xa9, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.DataType)
	if err != nil {
		return
	}
	// write "default_value"
	err = en.Append(0xad, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = z.DefaultValue.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "not_null"
	err = en.Append(0xa8, 0x6e, 0x6f, 0x74, 0x5f, 0x6e, 0x75, 0x6c, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.NotNull)
	if err != nil {
		return
	}
	// write "position"
	err = en.Append(0xa8, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt32(z.Position)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Column) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "name"
	o = append(o, 0x85, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "data_type"
	o = append(o, 0xa9, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.DataType)
	// string "default_value"
	o = append(o, 0xad, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o, err = z.DefaultValue.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "not_null"
	o = append(o, 0xa8, 0x6e, 0x6f, 0x74, 0x5f, 0x6e, 0x75, 0x6c, 0x6c)
	o = msgp.AppendBool(o, z.NotNull)
	// string "position"
	o = append(o, 0xa8, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt32(o, z.Position)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Column) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "data_type":
			z.DataType, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "default_value":
			bts, err = z.DefaultValue.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "not_null":
			z.NotNull, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "position":
			z.Position, bts, err = msgp.ReadInt32Bytes(bts)
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

func (z *Column) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Name) + 10 + msgp.StringPrefixSize + len(z.DataType) + 14 + z.DefaultValue.Msgsize() + 9 + msgp.BoolSize + 9 + msgp.Int32Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Constraint) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "constraint_def":
			z.ConstraintDef, err = dc.ReadString()
			if err != nil {
				return
			}
		case "columns":
			err = z.Columns.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "foreign_schema":
			err = z.ForeignSchema.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "foreign_table":
			err = z.ForeignTable.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "foreign_columns":
			err = z.ForeignColumns.DecodeMsg(dc)
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
func (z *Constraint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "name"
	err = en.Append(0x86, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "constraint_def"
	err = en.Append(0xae, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x74, 0x5f, 0x64, 0x65, 0x66)
	if err != nil {
		return err
	}
	err = en.WriteString(z.ConstraintDef)
	if err != nil {
		return
	}
	// write "columns"
	err = en.Append(0xa7, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = z.Columns.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "foreign_schema"
	err = en.Append(0xae, 0x66, 0x6f, 0x72, 0x65, 0x69, 0x67, 0x6e, 0x5f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61)
	if err != nil {
		return err
	}
	err = z.ForeignSchema.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "foreign_table"
	err = en.Append(0xad, 0x66, 0x6f, 0x72, 0x65, 0x69, 0x67, 0x6e, 0x5f, 0x74, 0x61, 0x62, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = z.ForeignTable.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "foreign_columns"
	err = en.Append(0xaf, 0x66, 0x6f, 0x72, 0x65, 0x69, 0x67, 0x6e, 0x5f, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = z.ForeignColumns.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Constraint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "name"
	o = append(o, 0x86, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "constraint_def"
	o = append(o, 0xae, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x74, 0x5f, 0x64, 0x65, 0x66)
	o = msgp.AppendString(o, z.ConstraintDef)
	// string "columns"
	o = append(o, 0xa7, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	o, err = z.Columns.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "foreign_schema"
	o = append(o, 0xae, 0x66, 0x6f, 0x72, 0x65, 0x69, 0x67, 0x6e, 0x5f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61)
	o, err = z.ForeignSchema.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "foreign_table"
	o = append(o, 0xad, 0x66, 0x6f, 0x72, 0x65, 0x69, 0x67, 0x6e, 0x5f, 0x74, 0x61, 0x62, 0x6c, 0x65)
	o, err = z.ForeignTable.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "foreign_columns"
	o = append(o, 0xaf, 0x66, 0x6f, 0x72, 0x65, 0x69, 0x67, 0x6e, 0x5f, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	o, err = z.ForeignColumns.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Constraint) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "constraint_def":
			z.ConstraintDef, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "columns":
			bts, err = z.Columns.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "foreign_schema":
			bts, err = z.ForeignSchema.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "foreign_table":
			bts, err = z.ForeignTable.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "foreign_columns":
			bts, err = z.ForeignColumns.UnmarshalMsg(bts)
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

func (z *Constraint) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Name) + 15 + msgp.StringPrefixSize + len(z.ConstraintDef) + 8 + z.Columns.Msgsize() + 15 + z.ForeignSchema.Msgsize() + 14 + z.ForeignTable.Msgsize() + 16 + z.ForeignColumns.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Index) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var ajw uint32
	ajw, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for ajw > 0 {
		ajw--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "columns":
			z.Columns, err = dc.ReadString()
			if err != nil {
				return
			}
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "size_bytes":
			z.SizeBytes, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "wasted_bytes":
			z.WastedBytes, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "is_primary":
			z.IsPrimary, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "is_unique":
			z.IsUnique, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "is_valid":
			z.IsValid, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "index_def":
			z.IndexDef, err = dc.ReadString()
			if err != nil {
				return
			}
		case "constraint_def":
			err = z.ConstraintDef.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_scan":
			err = z.IdxScan.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_tup_read":
			err = z.IdxTupRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_tup_fetch":
			err = z.IdxTupFetch.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_blks_read":
			err = z.IdxBlksRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_blks_hit":
			err = z.IdxBlksHit.DecodeMsg(dc)
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
func (z *Index) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "columns"
	err = en.Append(0x8e, 0xa7, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Columns)
	if err != nil {
		return
	}
	// write "name"
	err = en.Append(0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "size_bytes"
	err = en.Append(0xaa, 0x73, 0x69, 0x7a, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SizeBytes)
	if err != nil {
		return
	}
	// write "wasted_bytes"
	err = en.Append(0xac, 0x77, 0x61, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.WastedBytes)
	if err != nil {
		return
	}
	// write "is_primary"
	err = en.Append(0xaa, 0x69, 0x73, 0x5f, 0x70, 0x72, 0x69, 0x6d, 0x61, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsPrimary)
	if err != nil {
		return
	}
	// write "is_unique"
	err = en.Append(0xa9, 0x69, 0x73, 0x5f, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsUnique)
	if err != nil {
		return
	}
	// write "is_valid"
	err = en.Append(0xa8, 0x69, 0x73, 0x5f, 0x76, 0x61, 0x6c, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsValid)
	if err != nil {
		return
	}
	// write "index_def"
	err = en.Append(0xa9, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x5f, 0x64, 0x65, 0x66)
	if err != nil {
		return err
	}
	err = en.WriteString(z.IndexDef)
	if err != nil {
		return
	}
	// write "constraint_def"
	err = en.Append(0xae, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x74, 0x5f, 0x64, 0x65, 0x66)
	if err != nil {
		return err
	}
	err = z.ConstraintDef.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_scan"
	err = en.Append(0xa8, 0x69, 0x64, 0x78, 0x5f, 0x73, 0x63, 0x61, 0x6e)
	if err != nil {
		return err
	}
	err = z.IdxScan.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_tup_read"
	err = en.Append(0xac, 0x69, 0x64, 0x78, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.IdxTupRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_tup_fetch"
	err = en.Append(0xad, 0x69, 0x64, 0x78, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x66, 0x65, 0x74, 0x63, 0x68)
	if err != nil {
		return err
	}
	err = z.IdxTupFetch.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_blks_read"
	err = en.Append(0xad, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.IdxBlksRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_blks_hit"
	err = en.Append(0xac, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = z.IdxBlksHit.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Index) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 14
	// string "columns"
	o = append(o, 0x8e, 0xa7, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	o = msgp.AppendString(o, z.Columns)
	// string "name"
	o = append(o, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "size_bytes"
	o = append(o, 0xaa, 0x73, 0x69, 0x7a, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.SizeBytes)
	// string "wasted_bytes"
	o = append(o, 0xac, 0x77, 0x61, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.WastedBytes)
	// string "is_primary"
	o = append(o, 0xaa, 0x69, 0x73, 0x5f, 0x70, 0x72, 0x69, 0x6d, 0x61, 0x72, 0x79)
	o = msgp.AppendBool(o, z.IsPrimary)
	// string "is_unique"
	o = append(o, 0xa9, 0x69, 0x73, 0x5f, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65)
	o = msgp.AppendBool(o, z.IsUnique)
	// string "is_valid"
	o = append(o, 0xa8, 0x69, 0x73, 0x5f, 0x76, 0x61, 0x6c, 0x69, 0x64)
	o = msgp.AppendBool(o, z.IsValid)
	// string "index_def"
	o = append(o, 0xa9, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x5f, 0x64, 0x65, 0x66)
	o = msgp.AppendString(o, z.IndexDef)
	// string "constraint_def"
	o = append(o, 0xae, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x74, 0x5f, 0x64, 0x65, 0x66)
	o, err = z.ConstraintDef.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_scan"
	o = append(o, 0xa8, 0x69, 0x64, 0x78, 0x5f, 0x73, 0x63, 0x61, 0x6e)
	o, err = z.IdxScan.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_tup_read"
	o = append(o, 0xac, 0x69, 0x64, 0x78, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.IdxTupRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_tup_fetch"
	o = append(o, 0xad, 0x69, 0x64, 0x78, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x66, 0x65, 0x74, 0x63, 0x68)
	o, err = z.IdxTupFetch.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_blks_read"
	o = append(o, 0xad, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.IdxBlksRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_blks_hit"
	o = append(o, 0xac, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o, err = z.IdxBlksHit.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Index) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "columns":
			z.Columns, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "size_bytes":
			z.SizeBytes, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "wasted_bytes":
			z.WastedBytes, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "is_primary":
			z.IsPrimary, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "is_unique":
			z.IsUnique, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "is_valid":
			z.IsValid, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "index_def":
			z.IndexDef, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "constraint_def":
			bts, err = z.ConstraintDef.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_scan":
			bts, err = z.IdxScan.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_tup_read":
			bts, err = z.IdxTupRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_tup_fetch":
			bts, err = z.IdxTupFetch.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_blks_read":
			bts, err = z.IdxBlksRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_blks_hit":
			bts, err = z.IdxBlksHit.UnmarshalMsg(bts)
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

func (z *Index) Msgsize() (s int) {
	s = 1 + 8 + msgp.StringPrefixSize + len(z.Columns) + 5 + msgp.StringPrefixSize + len(z.Name) + 11 + msgp.Int64Size + 13 + msgp.Int64Size + 11 + msgp.BoolSize + 10 + msgp.BoolSize + 9 + msgp.BoolSize + 10 + msgp.StringPrefixSize + len(z.IndexDef) + 15 + z.ConstraintDef.Msgsize() + 9 + z.IdxScan.Msgsize() + 13 + z.IdxTupRead.Msgsize() + 14 + z.IdxTupFetch.Msgsize() + 14 + z.IdxBlksRead.Msgsize() + 13 + z.IdxBlksHit.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Oid) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var hct int64
		hct, err = dc.ReadInt64()
		(*z) = Oid(hct)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Oid) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteInt64(int64(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Oid) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt64(o, int64(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Oid) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var cua int64
		cua, bts, err = msgp.ReadInt64Bytes(bts)
		(*z) = Oid(cua)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

func (z Oid) Msgsize() (s int) {
	s = msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Relation) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var pks uint32
	pks, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for pks > 0 {
		pks--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "oid":
			{
				var jfb int64
				jfb, err = dc.ReadInt64()
				z.Oid = Oid(jfb)
			}
			if err != nil {
				return
			}
		case "schema_name":
			z.SchemaName, err = dc.ReadString()
			if err != nil {
				return
			}
		case "table_name":
			z.TableName, err = dc.ReadString()
			if err != nil {
				return
			}
		case "relation_type":
			z.RelationType, err = dc.ReadString()
			if err != nil {
				return
			}
		case "stats":
			err = z.Stats.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "columns":
			var cxo uint32
			cxo, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Columns) >= int(cxo) {
				z.Columns = z.Columns[:cxo]
			} else {
				z.Columns = make([]Column, cxo)
			}
			for xhx := range z.Columns {
				err = z.Columns[xhx].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "indices":
			var eff uint32
			eff, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Indices) >= int(eff) {
				z.Indices = z.Indices[:eff]
			} else {
				z.Indices = make([]Index, eff)
			}
			for lqf := range z.Indices {
				err = z.Indices[lqf].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "constraints":
			var rsw uint32
			rsw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Constraints) >= int(rsw) {
				z.Constraints = z.Constraints[:rsw]
			} else {
				z.Constraints = make([]Constraint, rsw)
			}
			for daf := range z.Constraints {
				err = z.Constraints[daf].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "view_definition":
			z.ViewDefinition, err = dc.ReadString()
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
func (z *Relation) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "oid"
	err = en.Append(0x89, 0xa3, 0x6f, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(int64(z.Oid))
	if err != nil {
		return
	}
	// write "schema_name"
	err = en.Append(0xab, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.SchemaName)
	if err != nil {
		return
	}
	// write "table_name"
	err = en.Append(0xaa, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.TableName)
	if err != nil {
		return
	}
	// write "relation_type"
	err = en.Append(0xad, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.RelationType)
	if err != nil {
		return
	}
	// write "stats"
	err = en.Append(0xa5, 0x73, 0x74, 0x61, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = z.Stats.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "columns"
	err = en.Append(0xa7, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Columns)))
	if err != nil {
		return
	}
	for xhx := range z.Columns {
		err = z.Columns[xhx].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "indices"
	err = en.Append(0xa7, 0x69, 0x6e, 0x64, 0x69, 0x63, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Indices)))
	if err != nil {
		return
	}
	for lqf := range z.Indices {
		err = z.Indices[lqf].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "constraints"
	err = en.Append(0xab, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Constraints)))
	if err != nil {
		return
	}
	for daf := range z.Constraints {
		err = z.Constraints[daf].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "view_definition"
	err = en.Append(0xaf, 0x76, 0x69, 0x65, 0x77, 0x5f, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteString(z.ViewDefinition)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Relation) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "oid"
	o = append(o, 0x89, 0xa3, 0x6f, 0x69, 0x64)
	o = msgp.AppendInt64(o, int64(z.Oid))
	// string "schema_name"
	o = append(o, 0xab, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.SchemaName)
	// string "table_name"
	o = append(o, 0xaa, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.TableName)
	// string "relation_type"
	o = append(o, 0xad, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.RelationType)
	// string "stats"
	o = append(o, 0xa5, 0x73, 0x74, 0x61, 0x74, 0x73)
	o, err = z.Stats.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "columns"
	o = append(o, 0xa7, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Columns)))
	for xhx := range z.Columns {
		o, err = z.Columns[xhx].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "indices"
	o = append(o, 0xa7, 0x69, 0x6e, 0x64, 0x69, 0x63, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Indices)))
	for lqf := range z.Indices {
		o, err = z.Indices[lqf].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "constraints"
	o = append(o, 0xab, 0x63, 0x6f, 0x6e, 0x73, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Constraints)))
	for daf := range z.Constraints {
		o, err = z.Constraints[daf].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "view_definition"
	o = append(o, 0xaf, 0x76, 0x69, 0x65, 0x77, 0x5f, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendString(o, z.ViewDefinition)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Relation) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var xpk uint32
	xpk, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for xpk > 0 {
		xpk--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "oid":
			{
				var dnj int64
				dnj, bts, err = msgp.ReadInt64Bytes(bts)
				z.Oid = Oid(dnj)
			}
			if err != nil {
				return
			}
		case "schema_name":
			z.SchemaName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "table_name":
			z.TableName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "relation_type":
			z.RelationType, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "stats":
			bts, err = z.Stats.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "columns":
			var obc uint32
			obc, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Columns) >= int(obc) {
				z.Columns = z.Columns[:obc]
			} else {
				z.Columns = make([]Column, obc)
			}
			for xhx := range z.Columns {
				bts, err = z.Columns[xhx].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "indices":
			var snv uint32
			snv, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Indices) >= int(snv) {
				z.Indices = z.Indices[:snv]
			} else {
				z.Indices = make([]Index, snv)
			}
			for lqf := range z.Indices {
				bts, err = z.Indices[lqf].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "constraints":
			var kgt uint32
			kgt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Constraints) >= int(kgt) {
				z.Constraints = z.Constraints[:kgt]
			} else {
				z.Constraints = make([]Constraint, kgt)
			}
			for daf := range z.Constraints {
				bts, err = z.Constraints[daf].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "view_definition":
			z.ViewDefinition, bts, err = msgp.ReadStringBytes(bts)
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

func (z *Relation) Msgsize() (s int) {
	s = 1 + 4 + msgp.Int64Size + 12 + msgp.StringPrefixSize + len(z.SchemaName) + 11 + msgp.StringPrefixSize + len(z.TableName) + 14 + msgp.StringPrefixSize + len(z.RelationType) + 6 + z.Stats.Msgsize() + 8 + msgp.ArrayHeaderSize
	for xhx := range z.Columns {
		s += z.Columns[xhx].Msgsize()
	}
	s += 8 + msgp.ArrayHeaderSize
	for lqf := range z.Indices {
		s += z.Indices[lqf].Msgsize()
	}
	s += 12 + msgp.ArrayHeaderSize
	for daf := range z.Constraints {
		s += z.Constraints[daf].Msgsize()
	}
	s += 16 + msgp.StringPrefixSize + len(z.ViewDefinition)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RelationStats) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var ema uint32
	ema, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for ema > 0 {
		ema--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "size_bytes":
			z.SizeBytes, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "wasted_bytes":
			z.WastedBytes, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "seq_scan":
			err = z.SeqScan.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "seq_tup_read":
			err = z.SeqTupRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_scan":
			err = z.IdxScan.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_tup_fetch":
			err = z.IdxTupFetch.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_tup_ins":
			err = z.NTupIns.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_tup_upd":
			err = z.NTupUpd.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_tup_del":
			err = z.NTupDel.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_tup_hot_upd":
			err = z.NTupHotUpd.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_live_tup":
			err = z.NLiveTup.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_dead_tup":
			err = z.NDeadTup.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "n_mod_since_analyze":
			err = z.NModSinceAnalyze.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "last_vacuum":
			err = z.LastVacuum.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "last_autovacuum":
			err = z.LastAutovacuum.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "last_analyze":
			err = z.LastAnalyze.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "last_autoanalyze":
			err = z.LastAutoanalyze.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "vacuum_count":
			err = z.VacuumCount.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "autovacuum_count":
			err = z.AutovacuumCount.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "analyze_count":
			err = z.AnalyzeCount.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "autoanalyze_count":
			err = z.AutoanalyzeCount.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "heap_blks_read":
			err = z.HeapBlksRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "heap_blks_hit":
			err = z.HeapBlksHit.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_blks_read":
			err = z.IdxBlksRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "idx_blks_hit":
			err = z.IdxBlksHit.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "toast_blks_read":
			err = z.ToastBlksRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "toast_blks_hit":
			err = z.ToastBlksHit.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "tidx_blks_read":
			err = z.TidxBlksRead.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "tidx_blks_hit":
			err = z.TidxBlksHit.DecodeMsg(dc)
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
func (z *RelationStats) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 29
	// write "size_bytes"
	err = en.Append(0xde, 0x0, 0x1d, 0xaa, 0x73, 0x69, 0x7a, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SizeBytes)
	if err != nil {
		return
	}
	// write "wasted_bytes"
	err = en.Append(0xac, 0x77, 0x61, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.WastedBytes)
	if err != nil {
		return
	}
	// write "seq_scan"
	err = en.Append(0xa8, 0x73, 0x65, 0x71, 0x5f, 0x73, 0x63, 0x61, 0x6e)
	if err != nil {
		return err
	}
	err = z.SeqScan.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "seq_tup_read"
	err = en.Append(0xac, 0x73, 0x65, 0x71, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.SeqTupRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_scan"
	err = en.Append(0xa8, 0x69, 0x64, 0x78, 0x5f, 0x73, 0x63, 0x61, 0x6e)
	if err != nil {
		return err
	}
	err = z.IdxScan.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_tup_fetch"
	err = en.Append(0xad, 0x69, 0x64, 0x78, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x66, 0x65, 0x74, 0x63, 0x68)
	if err != nil {
		return err
	}
	err = z.IdxTupFetch.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_tup_ins"
	err = en.Append(0xa9, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x69, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = z.NTupIns.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_tup_upd"
	err = en.Append(0xa9, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x75, 0x70, 0x64)
	if err != nil {
		return err
	}
	err = z.NTupUpd.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_tup_del"
	err = en.Append(0xa9, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x64, 0x65, 0x6c)
	if err != nil {
		return err
	}
	err = z.NTupDel.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_tup_hot_upd"
	err = en.Append(0xad, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x68, 0x6f, 0x74, 0x5f, 0x75, 0x70, 0x64)
	if err != nil {
		return err
	}
	err = z.NTupHotUpd.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_live_tup"
	err = en.Append(0xaa, 0x6e, 0x5f, 0x6c, 0x69, 0x76, 0x65, 0x5f, 0x74, 0x75, 0x70)
	if err != nil {
		return err
	}
	err = z.NLiveTup.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_dead_tup"
	err = en.Append(0xaa, 0x6e, 0x5f, 0x64, 0x65, 0x61, 0x64, 0x5f, 0x74, 0x75, 0x70)
	if err != nil {
		return err
	}
	err = z.NDeadTup.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "n_mod_since_analyze"
	err = en.Append(0xb3, 0x6e, 0x5f, 0x6d, 0x6f, 0x64, 0x5f, 0x73, 0x69, 0x6e, 0x63, 0x65, 0x5f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65)
	if err != nil {
		return err
	}
	err = z.NModSinceAnalyze.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "last_vacuum"
	err = en.Append(0xab, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = z.LastVacuum.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "last_autovacuum"
	err = en.Append(0xaf, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x61, 0x75, 0x74, 0x6f, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = z.LastAutovacuum.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "last_analyze"
	err = en.Append(0xac, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65)
	if err != nil {
		return err
	}
	err = z.LastAnalyze.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "last_autoanalyze"
	err = en.Append(0xb0, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x61, 0x75, 0x74, 0x6f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65)
	if err != nil {
		return err
	}
	err = z.LastAutoanalyze.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "vacuum_count"
	err = en.Append(0xac, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = z.VacuumCount.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "autovacuum_count"
	err = en.Append(0xb0, 0x61, 0x75, 0x74, 0x6f, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = z.AutovacuumCount.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "analyze_count"
	err = en.Append(0xad, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = z.AnalyzeCount.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "autoanalyze_count"
	err = en.Append(0xb1, 0x61, 0x75, 0x74, 0x6f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = z.AutoanalyzeCount.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "heap_blks_read"
	err = en.Append(0xae, 0x68, 0x65, 0x61, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.HeapBlksRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "heap_blks_hit"
	err = en.Append(0xad, 0x68, 0x65, 0x61, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = z.HeapBlksHit.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_blks_read"
	err = en.Append(0xad, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.IdxBlksRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "idx_blks_hit"
	err = en.Append(0xac, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = z.IdxBlksHit.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "toast_blks_read"
	err = en.Append(0xaf, 0x74, 0x6f, 0x61, 0x73, 0x74, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.ToastBlksRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "toast_blks_hit"
	err = en.Append(0xae, 0x74, 0x6f, 0x61, 0x73, 0x74, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = z.ToastBlksHit.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "tidx_blks_read"
	err = en.Append(0xae, 0x74, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	if err != nil {
		return err
	}
	err = z.TidxBlksRead.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "tidx_blks_hit"
	err = en.Append(0xad, 0x74, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = z.TidxBlksHit.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RelationStats) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 29
	// string "size_bytes"
	o = append(o, 0xde, 0x0, 0x1d, 0xaa, 0x73, 0x69, 0x7a, 0x65, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.SizeBytes)
	// string "wasted_bytes"
	o = append(o, 0xac, 0x77, 0x61, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.WastedBytes)
	// string "seq_scan"
	o = append(o, 0xa8, 0x73, 0x65, 0x71, 0x5f, 0x73, 0x63, 0x61, 0x6e)
	o, err = z.SeqScan.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "seq_tup_read"
	o = append(o, 0xac, 0x73, 0x65, 0x71, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.SeqTupRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_scan"
	o = append(o, 0xa8, 0x69, 0x64, 0x78, 0x5f, 0x73, 0x63, 0x61, 0x6e)
	o, err = z.IdxScan.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_tup_fetch"
	o = append(o, 0xad, 0x69, 0x64, 0x78, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x66, 0x65, 0x74, 0x63, 0x68)
	o, err = z.IdxTupFetch.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_tup_ins"
	o = append(o, 0xa9, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x69, 0x6e, 0x73)
	o, err = z.NTupIns.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_tup_upd"
	o = append(o, 0xa9, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x75, 0x70, 0x64)
	o, err = z.NTupUpd.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_tup_del"
	o = append(o, 0xa9, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x64, 0x65, 0x6c)
	o, err = z.NTupDel.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_tup_hot_upd"
	o = append(o, 0xad, 0x6e, 0x5f, 0x74, 0x75, 0x70, 0x5f, 0x68, 0x6f, 0x74, 0x5f, 0x75, 0x70, 0x64)
	o, err = z.NTupHotUpd.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_live_tup"
	o = append(o, 0xaa, 0x6e, 0x5f, 0x6c, 0x69, 0x76, 0x65, 0x5f, 0x74, 0x75, 0x70)
	o, err = z.NLiveTup.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_dead_tup"
	o = append(o, 0xaa, 0x6e, 0x5f, 0x64, 0x65, 0x61, 0x64, 0x5f, 0x74, 0x75, 0x70)
	o, err = z.NDeadTup.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "n_mod_since_analyze"
	o = append(o, 0xb3, 0x6e, 0x5f, 0x6d, 0x6f, 0x64, 0x5f, 0x73, 0x69, 0x6e, 0x63, 0x65, 0x5f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65)
	o, err = z.NModSinceAnalyze.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "last_vacuum"
	o = append(o, 0xab, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d)
	o, err = z.LastVacuum.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "last_autovacuum"
	o = append(o, 0xaf, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x61, 0x75, 0x74, 0x6f, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d)
	o, err = z.LastAutovacuum.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "last_analyze"
	o = append(o, 0xac, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65)
	o, err = z.LastAnalyze.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "last_autoanalyze"
	o = append(o, 0xb0, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x61, 0x75, 0x74, 0x6f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65)
	o, err = z.LastAutoanalyze.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "vacuum_count"
	o = append(o, 0xac, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	o, err = z.VacuumCount.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "autovacuum_count"
	o = append(o, 0xb0, 0x61, 0x75, 0x74, 0x6f, 0x76, 0x61, 0x63, 0x75, 0x75, 0x6d, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	o, err = z.AutovacuumCount.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "analyze_count"
	o = append(o, 0xad, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	o, err = z.AnalyzeCount.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "autoanalyze_count"
	o = append(o, 0xb1, 0x61, 0x75, 0x74, 0x6f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x7a, 0x65, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	o, err = z.AutoanalyzeCount.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "heap_blks_read"
	o = append(o, 0xae, 0x68, 0x65, 0x61, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.HeapBlksRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "heap_blks_hit"
	o = append(o, 0xad, 0x68, 0x65, 0x61, 0x70, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o, err = z.HeapBlksHit.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_blks_read"
	o = append(o, 0xad, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.IdxBlksRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "idx_blks_hit"
	o = append(o, 0xac, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o, err = z.IdxBlksHit.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "toast_blks_read"
	o = append(o, 0xaf, 0x74, 0x6f, 0x61, 0x73, 0x74, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.ToastBlksRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "toast_blks_hit"
	o = append(o, 0xae, 0x74, 0x6f, 0x61, 0x73, 0x74, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o, err = z.ToastBlksHit.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "tidx_blks_read"
	o = append(o, 0xae, 0x74, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x72, 0x65, 0x61, 0x64)
	o, err = z.TidxBlksRead.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "tidx_blks_hit"
	o = append(o, 0xad, 0x74, 0x69, 0x64, 0x78, 0x5f, 0x62, 0x6c, 0x6b, 0x73, 0x5f, 0x68, 0x69, 0x74)
	o, err = z.TidxBlksHit.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RelationStats) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var pez uint32
	pez, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for pez > 0 {
		pez--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "size_bytes":
			z.SizeBytes, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "wasted_bytes":
			z.WastedBytes, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "seq_scan":
			bts, err = z.SeqScan.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "seq_tup_read":
			bts, err = z.SeqTupRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_scan":
			bts, err = z.IdxScan.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_tup_fetch":
			bts, err = z.IdxTupFetch.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_tup_ins":
			bts, err = z.NTupIns.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_tup_upd":
			bts, err = z.NTupUpd.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_tup_del":
			bts, err = z.NTupDel.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_tup_hot_upd":
			bts, err = z.NTupHotUpd.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_live_tup":
			bts, err = z.NLiveTup.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_dead_tup":
			bts, err = z.NDeadTup.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "n_mod_since_analyze":
			bts, err = z.NModSinceAnalyze.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "last_vacuum":
			bts, err = z.LastVacuum.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "last_autovacuum":
			bts, err = z.LastAutovacuum.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "last_analyze":
			bts, err = z.LastAnalyze.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "last_autoanalyze":
			bts, err = z.LastAutoanalyze.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "vacuum_count":
			bts, err = z.VacuumCount.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "autovacuum_count":
			bts, err = z.AutovacuumCount.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "analyze_count":
			bts, err = z.AnalyzeCount.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "autoanalyze_count":
			bts, err = z.AutoanalyzeCount.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "heap_blks_read":
			bts, err = z.HeapBlksRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "heap_blks_hit":
			bts, err = z.HeapBlksHit.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_blks_read":
			bts, err = z.IdxBlksRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "idx_blks_hit":
			bts, err = z.IdxBlksHit.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "toast_blks_read":
			bts, err = z.ToastBlksRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "toast_blks_hit":
			bts, err = z.ToastBlksHit.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "tidx_blks_read":
			bts, err = z.TidxBlksRead.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "tidx_blks_hit":
			bts, err = z.TidxBlksHit.UnmarshalMsg(bts)
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

func (z *RelationStats) Msgsize() (s int) {
	s = 3 + 11 + msgp.Int64Size + 13 + msgp.Int64Size + 9 + z.SeqScan.Msgsize() + 13 + z.SeqTupRead.Msgsize() + 9 + z.IdxScan.Msgsize() + 14 + z.IdxTupFetch.Msgsize() + 10 + z.NTupIns.Msgsize() + 10 + z.NTupUpd.Msgsize() + 10 + z.NTupDel.Msgsize() + 14 + z.NTupHotUpd.Msgsize() + 11 + z.NLiveTup.Msgsize() + 11 + z.NDeadTup.Msgsize() + 20 + z.NModSinceAnalyze.Msgsize() + 12 + z.LastVacuum.Msgsize() + 16 + z.LastAutovacuum.Msgsize() + 13 + z.LastAnalyze.Msgsize() + 17 + z.LastAutoanalyze.Msgsize() + 13 + z.VacuumCount.Msgsize() + 17 + z.AutovacuumCount.Msgsize() + 14 + z.AnalyzeCount.Msgsize() + 18 + z.AutoanalyzeCount.Msgsize() + 15 + z.HeapBlksRead.Msgsize() + 14 + z.HeapBlksHit.Msgsize() + 14 + z.IdxBlksRead.Msgsize() + 13 + z.IdxBlksHit.Msgsize() + 16 + z.ToastBlksRead.Msgsize() + 15 + z.ToastBlksHit.Msgsize() + 15 + z.TidxBlksRead.Msgsize() + 14 + z.TidxBlksHit.Msgsize()
	return
}
