//go:generate msgp

package snapshot

type Statement struct {
	Userid            int     `msg:"userid"`
	Query             string  `msg:"query"`
	Calls             int64   `msg:"calls"`
	TotalTime         float64 `msg:"total_time"`
	Rows              int64   `msg:"rows"`
	SharedBlksHit     int64   `msg:"shared_blks_hit"`
	SharedBlksRead    int64   `msg:"shared_blks_read"`
	SharedBlksDirtied int64   `msg:"shared_blks_dirtied"`
	SharedBlksWritten int64   `msg:"shared_blks_written"`
	LocalBlksHit      int64   `msg:"local_blks_hit"`
	LocalBlksRead     int64   `msg:"local_blks_read"`
	LocalBlksDirtied  int64   `msg:"local_blks_dirtied"`
	LocalBlksWritten  int64   `msg:"local_blks_written"`
	TempBlksRead      int64   `msg:"temp_blks_read"`
	TempBlksWritten   int64   `msg:"temp_blks_written"`
	BlkReadTime       float64 `msg:"blk_read_time"`
	BlkWriteTime      float64 `msg:"blk_write_time"`

	// Postgres 9.4+
	Queryid int64 `msg:"queryid"`

	// Postgres 9.5+
	MinTime    float64 `msg:"min_time"`
	MaxTime    float64 `msg:"max_time"`
	MeanTime   float64 `msg:"mean_time"`
	StddevTime float64 `msg:"stddev_time"`
}

func (curr Statement) DiffSince(prev Statement) Statement {
	return Statement{
		Userid:            curr.Userid,
		Query:             curr.Query,
		Queryid:           curr.Queryid,
		Calls:             curr.Calls - prev.Calls,
		TotalTime:         curr.TotalTime - prev.TotalTime,
		Rows:              curr.Rows - prev.Rows,
		SharedBlksHit:     curr.SharedBlksHit - prev.SharedBlksHit,
		SharedBlksRead:    curr.SharedBlksRead - prev.SharedBlksRead,
		SharedBlksDirtied: curr.SharedBlksDirtied - prev.SharedBlksDirtied,
		SharedBlksWritten: curr.SharedBlksWritten - prev.SharedBlksWritten,
		LocalBlksHit:      curr.LocalBlksHit - prev.LocalBlksHit,
		LocalBlksRead:     curr.LocalBlksRead - prev.LocalBlksRead,
		LocalBlksDirtied:  curr.LocalBlksDirtied - prev.LocalBlksDirtied,
		LocalBlksWritten:  curr.LocalBlksWritten - prev.LocalBlksWritten,
		TempBlksRead:      curr.TempBlksRead - prev.TempBlksRead,
		TempBlksWritten:   curr.TempBlksWritten - prev.TempBlksWritten,
		BlkReadTime:       curr.BlkReadTime - prev.BlkReadTime,
		BlkWriteTime:      curr.BlkWriteTime - prev.BlkWriteTime,
	}
}
