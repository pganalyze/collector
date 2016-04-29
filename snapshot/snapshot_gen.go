package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Snapshot) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "backends":
			var wht uint32
			wht, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.ActiveQueries) >= int(wht) {
				z.ActiveQueries = z.ActiveQueries[:wht]
			} else {
				z.ActiveQueries = make([]Activity, wht)
			}
			for xvk := range z.ActiveQueries {
				err = z.ActiveQueries[xvk].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "queries":
			var hct uint32
			hct, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Statements) >= int(hct) {
				z.Statements = z.Statements[:hct]
			} else {
				z.Statements = make([]Statement, hct)
			}
			for bzg := range z.Statements {
				err = z.Statements[bzg].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "postgres":
			err = z.Postgres.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "system":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.System = nil
			} else {
				if z.System == nil {
					z.System = new(System)
				}
				err = z.System.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "logs":
			var cua uint32
			cua, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Logs) >= int(cua) {
				z.Logs = z.Logs[:cua]
			} else {
				z.Logs = make([]LogLine, cua)
			}
			for bai := range z.Logs {
				err = z.Logs[bai].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "explains":
			var xhx uint32
			xhx, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Explains) >= int(xhx) {
				z.Explains = z.Explains[:xhx]
			} else {
				z.Explains = make([]Explain, xhx)
			}
			for cmr := range z.Explains {
				err = z.Explains[cmr].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "opts":
			var lqf uint32
			lqf, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			for lqf > 0 {
				lqf--
				field, err = dc.ReadMapKeyPtr()
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "statement_stats_are_diffed":
					z.Opts.StatementStatsAreDiffed, err = dc.ReadBool()
					if err != nil {
						return
					}
				case "postgres_relation_stats_are_diffed":
					z.Opts.PostgresRelationStatsAreDiffed, err = dc.ReadBool()
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
func (z *Snapshot) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "backends"
	err = en.Append(0x87, 0xa8, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.ActiveQueries)))
	if err != nil {
		return
	}
	for xvk := range z.ActiveQueries {
		err = z.ActiveQueries[xvk].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "queries"
	err = en.Append(0xa7, 0x71, 0x75, 0x65, 0x72, 0x69, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Statements)))
	if err != nil {
		return
	}
	for bzg := range z.Statements {
		err = z.Statements[bzg].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "postgres"
	err = en.Append(0xa8, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.Postgres.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "system"
	err = en.Append(0xa6, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d)
	if err != nil {
		return err
	}
	if z.System == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.System.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "logs"
	err = en.Append(0xa4, 0x6c, 0x6f, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Logs)))
	if err != nil {
		return
	}
	for bai := range z.Logs {
		err = z.Logs[bai].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "explains"
	err = en.Append(0xa8, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x69, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Explains)))
	if err != nil {
		return
	}
	for cmr := range z.Explains {
		err = z.Explains[cmr].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "opts"
	// map header, size 2
	// write "statement_stats_are_diffed"
	err = en.Append(0xa4, 0x6f, 0x70, 0x74, 0x73, 0x82, 0xba, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Opts.StatementStatsAreDiffed)
	if err != nil {
		return
	}
	// write "postgres_relation_stats_are_diffed"
	err = en.Append(0xd9, 0x22, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x5f, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.Opts.PostgresRelationStatsAreDiffed)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Snapshot) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "backends"
	o = append(o, 0x87, 0xa8, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ActiveQueries)))
	for xvk := range z.ActiveQueries {
		o, err = z.ActiveQueries[xvk].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "queries"
	o = append(o, 0xa7, 0x71, 0x75, 0x65, 0x72, 0x69, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Statements)))
	for bzg := range z.Statements {
		o, err = z.Statements[bzg].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "postgres"
	o = append(o, 0xa8, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73)
	o, err = z.Postgres.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "system"
	o = append(o, 0xa6, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d)
	if z.System == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.System.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "logs"
	o = append(o, 0xa4, 0x6c, 0x6f, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Logs)))
	for bai := range z.Logs {
		o, err = z.Logs[bai].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "explains"
	o = append(o, 0xa8, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x69, 0x6e, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Explains)))
	for cmr := range z.Explains {
		o, err = z.Explains[cmr].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "opts"
	// map header, size 2
	// string "statement_stats_are_diffed"
	o = append(o, 0xa4, 0x6f, 0x70, 0x74, 0x73, 0x82, 0xba, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	o = msgp.AppendBool(o, z.Opts.StatementStatsAreDiffed)
	// string "postgres_relation_stats_are_diffed"
	o = append(o, 0xd9, 0x22, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x5f, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	o = msgp.AppendBool(o, z.Opts.PostgresRelationStatsAreDiffed)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Snapshot) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var daf uint32
	daf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for daf > 0 {
		daf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "backends":
			var pks uint32
			pks, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.ActiveQueries) >= int(pks) {
				z.ActiveQueries = z.ActiveQueries[:pks]
			} else {
				z.ActiveQueries = make([]Activity, pks)
			}
			for xvk := range z.ActiveQueries {
				bts, err = z.ActiveQueries[xvk].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "queries":
			var jfb uint32
			jfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Statements) >= int(jfb) {
				z.Statements = z.Statements[:jfb]
			} else {
				z.Statements = make([]Statement, jfb)
			}
			for bzg := range z.Statements {
				bts, err = z.Statements[bzg].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "postgres":
			bts, err = z.Postgres.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "system":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.System = nil
			} else {
				if z.System == nil {
					z.System = new(System)
				}
				bts, err = z.System.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "logs":
			var cxo uint32
			cxo, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Logs) >= int(cxo) {
				z.Logs = z.Logs[:cxo]
			} else {
				z.Logs = make([]LogLine, cxo)
			}
			for bai := range z.Logs {
				bts, err = z.Logs[bai].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "explains":
			var eff uint32
			eff, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Explains) >= int(eff) {
				z.Explains = z.Explains[:eff]
			} else {
				z.Explains = make([]Explain, eff)
			}
			for cmr := range z.Explains {
				bts, err = z.Explains[cmr].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "opts":
			var rsw uint32
			rsw, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			for rsw > 0 {
				rsw--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "statement_stats_are_diffed":
					z.Opts.StatementStatsAreDiffed, bts, err = msgp.ReadBoolBytes(bts)
					if err != nil {
						return
					}
				case "postgres_relation_stats_are_diffed":
					z.Opts.PostgresRelationStatsAreDiffed, bts, err = msgp.ReadBoolBytes(bts)
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

func (z *Snapshot) Msgsize() (s int) {
	s = 1 + 9 + msgp.ArrayHeaderSize
	for xvk := range z.ActiveQueries {
		s += z.ActiveQueries[xvk].Msgsize()
	}
	s += 8 + msgp.ArrayHeaderSize
	for bzg := range z.Statements {
		s += z.Statements[bzg].Msgsize()
	}
	s += 9 + z.Postgres.Msgsize() + 7
	if z.System == nil {
		s += msgp.NilSize
	} else {
		s += z.System.Msgsize()
	}
	s += 5 + msgp.ArrayHeaderSize
	for bai := range z.Logs {
		s += z.Logs[bai].Msgsize()
	}
	s += 9 + msgp.ArrayHeaderSize
	for cmr := range z.Explains {
		s += z.Explains[cmr].Msgsize()
	}
	s += 5 + 1 + 27 + msgp.BoolSize + 36 + msgp.BoolSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SnapshotOpts) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var xpk uint32
	xpk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for xpk > 0 {
		xpk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "statement_stats_are_diffed":
			z.StatementStatsAreDiffed, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "postgres_relation_stats_are_diffed":
			z.PostgresRelationStatsAreDiffed, err = dc.ReadBool()
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
func (z SnapshotOpts) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "statement_stats_are_diffed"
	err = en.Append(0x82, 0xba, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.StatementStatsAreDiffed)
	if err != nil {
		return
	}
	// write "postgres_relation_stats_are_diffed"
	err = en.Append(0xd9, 0x22, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x5f, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.PostgresRelationStatsAreDiffed)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z SnapshotOpts) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "statement_stats_are_diffed"
	o = append(o, 0x82, 0xba, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	o = msgp.AppendBool(o, z.StatementStatsAreDiffed)
	// string "postgres_relation_stats_are_diffed"
	o = append(o, 0xd9, 0x22, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x5f, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x73, 0x5f, 0x61, 0x72, 0x65, 0x5f, 0x64, 0x69, 0x66, 0x66, 0x65, 0x64)
	o = msgp.AppendBool(o, z.PostgresRelationStatsAreDiffed)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SnapshotOpts) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var dnj uint32
	dnj, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for dnj > 0 {
		dnj--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "statement_stats_are_diffed":
			z.StatementStatsAreDiffed, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "postgres_relation_stats_are_diffed":
			z.PostgresRelationStatsAreDiffed, bts, err = msgp.ReadBoolBytes(bts)
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

func (z SnapshotOpts) Msgsize() (s int) {
	s = 1 + 27 + msgp.BoolSize + 36 + msgp.BoolSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SnapshotPostgres) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "schema":
			var pez uint32
			pez, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Relations) >= int(pez) {
				z.Relations = z.Relations[:pez]
			} else {
				z.Relations = make([]Relation, pez)
			}
			for obc := range z.Relations {
				err = z.Relations[obc].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "settings":
			var qke uint32
			qke, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Settings) >= int(qke) {
				z.Settings = z.Settings[:qke]
			} else {
				z.Settings = make([]Setting, qke)
			}
			for snv := range z.Settings {
				err = z.Settings[snv].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "functions":
			var qyh uint32
			qyh, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Functions) >= int(qyh) {
				z.Functions = z.Functions[:qyh]
			} else {
				z.Functions = make([]Function, qyh)
			}
			for kgt := range z.Functions {
				err = z.Functions[kgt].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "version":
			err = z.Version.DecodeMsg(dc)
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
func (z *SnapshotPostgres) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "schema"
	err = en.Append(0x84, 0xa6, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Relations)))
	if err != nil {
		return
	}
	for obc := range z.Relations {
		err = z.Relations[obc].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "settings"
	err = en.Append(0xa8, 0x73, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Settings)))
	if err != nil {
		return
	}
	for snv := range z.Settings {
		err = z.Settings[snv].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "functions"
	err = en.Append(0xa9, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Functions)))
	if err != nil {
		return
	}
	for kgt := range z.Functions {
		err = z.Functions[kgt].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "version"
	err = en.Append(0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = z.Version.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SnapshotPostgres) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "schema"
	o = append(o, 0x84, 0xa6, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Relations)))
	for obc := range z.Relations {
		o, err = z.Relations[obc].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "settings"
	o = append(o, 0xa8, 0x73, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Settings)))
	for snv := range z.Settings {
		o, err = z.Settings[snv].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "functions"
	o = append(o, 0xa9, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Functions)))
	for kgt := range z.Functions {
		o, err = z.Functions[kgt].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "version"
	o = append(o, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o, err = z.Version.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SnapshotPostgres) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var yzr uint32
	yzr, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for yzr > 0 {
		yzr--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "schema":
			var ywj uint32
			ywj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Relations) >= int(ywj) {
				z.Relations = z.Relations[:ywj]
			} else {
				z.Relations = make([]Relation, ywj)
			}
			for obc := range z.Relations {
				bts, err = z.Relations[obc].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "settings":
			var jpj uint32
			jpj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Settings) >= int(jpj) {
				z.Settings = z.Settings[:jpj]
			} else {
				z.Settings = make([]Setting, jpj)
			}
			for snv := range z.Settings {
				bts, err = z.Settings[snv].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "functions":
			var zpf uint32
			zpf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Functions) >= int(zpf) {
				z.Functions = z.Functions[:zpf]
			} else {
				z.Functions = make([]Function, zpf)
			}
			for kgt := range z.Functions {
				bts, err = z.Functions[kgt].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "version":
			bts, err = z.Version.UnmarshalMsg(bts)
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

func (z *SnapshotPostgres) Msgsize() (s int) {
	s = 1 + 7 + msgp.ArrayHeaderSize
	for obc := range z.Relations {
		s += z.Relations[obc].Msgsize()
	}
	s += 9 + msgp.ArrayHeaderSize
	for snv := range z.Settings {
		s += z.Settings[snv].Msgsize()
	}
	s += 10 + msgp.ArrayHeaderSize
	for kgt := range z.Functions {
		s += z.Functions[kgt].Msgsize()
	}
	s += 8 + z.Version.Msgsize()
	return
}
