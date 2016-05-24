package state

import "gopkg.in/guregu/null.v3"

type PostgresStatement struct {
	Userid            int     `json:"userid"`
	Query             string  `json:"query"`
	Calls             int64   `json:"calls"`
	TotalTime         float64 `json:"total_time"`
	Rows              int64   `json:"rows"`
	SharedBlksHit     int64   `json:"shared_blks_hit"`
	SharedBlksRead    int64   `json:"shared_blks_read"`
	SharedBlksDirtied int64   `json:"shared_blks_dirtied"`
	SharedBlksWritten int64   `json:"shared_blks_written"`
	LocalBlksHit      int64   `json:"local_blks_hit"`
	LocalBlksRead     int64   `json:"local_blks_read"`
	LocalBlksDirtied  int64   `json:"local_blks_dirtied"`
	LocalBlksWritten  int64   `json:"local_blks_written"`
	TempBlksRead      int64   `json:"temp_blks_read"`
	TempBlksWritten   int64   `json:"temp_blks_written"`
	BlkReadTime       float64 `json:"blk_read_time"`
	BlkWriteTime      float64 `json:"blk_write_time"`

	// Postgres 9.4+
	Queryid null.Int `json:"queryid"`

	// Postgres 9.5+
	MinTime    null.Float `json:"min_time"`
	MaxTime    null.Float `json:"max_time"`
	MeanTime   null.Float `json:"mean_time"`
	StddevTime null.Float `json:"stddev_time"`
}

func (curr PostgresStatement) DiffSince(prev PostgresStatement) PostgresStatement {
	return PostgresStatement{
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
