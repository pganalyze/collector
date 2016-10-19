package reports

import (
	"github.com/golang/protobuf/proto"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type BloatReport struct {
}

func (report *BloatReport) Run(server state.Server, logger *util.Logger) error {
	return nil
}

func (report *BloatReport) Result() proto.Message {
	return &pganalyze_collector.BloatReport{ReportRunUuid: "dummy"}
}
