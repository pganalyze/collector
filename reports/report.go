package reports

import (
	"github.com/golang/protobuf/proto"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type Report interface {
	Run(server state.Server, logger *util.Logger) error
	Result() proto.Message
}
