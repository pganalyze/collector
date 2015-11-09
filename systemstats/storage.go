package systemstats

import (
	null "gopkg.in/guregu/null.v2"
)

type Storage struct {
  BytesAvailable float64 `json:"bytes_available"`
  BytesTotal float64 `json:"bytes_total"`
  Mountpoint null.String `json:"mountpoint"`
  Name null.String `json:"name"`
  Path null.String `json:"path"`
  Perfdata StoragePerfdata `json:"perfdata"`
}

type StoragePerfdata struct {
  RdIos float64 `json:"rd_ios"`
  RdMerges float64 `json:"rd_merges"`
  RdSectors float64 `json:"rd_sectors"`
  RdTicks float64 `json:"rd_ticks"`
  WrIos float64 `json:"wr_ios"`
  WrMerges float64 `json:"wr_merges"`
  WrSectors float64 `json:"wr_sectors"`
  WrTicks float64 `json:"wr_ticks"`
  IosInProg float64 `json:"ios_in_prog"`
  TotTicks float64 `json:"tot_ticks"`
  RqTicks float64 `json:"rq_ticks"`
}
