package state

type CollectionStateCode int

const (
	CollectionStateUnchecked CollectionStateCode = iota
	CollectionStateNotAvailable
	CollectionStateError
	CollectionStateOkay
)

type CollectionState struct {
	State CollectionStateCode
	Msg   string
}

// summary should show, for each server (preceded by green ✓ or red ✗):
//  - detected system type / platform / id
//  - can collect system information? (or that not available on given system, or remote host specified and how to override)
//  - can connect to monitoring database?
//  - can access pg_stat_statements? (if yes, but old version, show error here)
//  - can collect schema information? (if not, which databases we could not monitor)
//  - can collect column stats? (if not, which databases have errors: first three with " and x more" or all with --verbose)
//  - can collect extended stats? (same as above: if not, which databases have errors)
//  - can collect log information? (whether disabled, and if not, status and how to disable, at least for Production plans)
//  - can collect explain plans?
//  - can use index advisor?

type DbCollectionState struct {
	CollectionState
	DbName string
}

type SelfCheckStatus struct {
	CollectionEnabled      CollectionState
	CollectorStatus        CollectionState
	SystemStats            CollectionState
	MonitoringDbConnection CollectionState
	PgStatStatements       CollectionState
	SchemaInformation      []DbCollectionState
	ColumnStats            []DbCollectionState
	ExtendedStats          []DbCollectionState
	LogInsights            CollectionState
	AutomatedExplain       CollectionState
}

// collection

func (s *Server) SelfCheckMarkCollectionOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.CollectionEnabled.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.CollectionEnabled.State = CollectionStateOkay
	s.SelfCheck.CollectionEnabled.Msg = "ok"
}

func (s *Server) SelfCheckMarkCollectionNotAvailable(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.CollectionEnabled.State = CollectionStateNotAvailable
	s.SelfCheck.CollectionEnabled.Msg = msg
}

// collector status

func (s *Server) SelfCheckMarkCollectorStatusOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.CollectorStatus.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.CollectorStatus.State = CollectionStateOkay
	s.SelfCheck.CollectorStatus.Msg = "ok"
}

func (s *Server) SelfCheckMarkCollectorStatusError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.CollectorStatus.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.CollectorStatus.State = CollectionStateError
	s.SelfCheck.CollectorStatus.Msg = msg
}

// system stats

func (s *Server) SelfCheckMarkSystemStatsOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.SystemStats.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.SystemStats.State = CollectionStateOkay
	s.SelfCheck.SystemStats.Msg = "ok"
}

func (s *Server) SelfCheckMarkSystemStatsNotAvailable() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SystemStats.State = CollectionStateNotAvailable
	s.SelfCheck.SystemStats.Msg = "not available on this platform"
}

func (s *Server) SelfCheckMarkSystemStatsError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SystemStats.State = CollectionStateNotAvailable
	s.SelfCheck.SystemStats.Msg = msg
}

// monitoring DB connection

func (s *Server) SelfCheckMarkMonitoringDbConnectionOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.MonitoringDbConnection.State = CollectionStateOkay
	s.SelfCheck.MonitoringDbConnection.Msg = "ok"
}

func (s *Server) SelfCheckMarkMonitoringDbConnectionError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.MonitoringDbConnection.State = CollectionStateNotAvailable
	s.SelfCheck.MonitoringDbConnection.Msg = msg
}

// pg_stat_statements

func (s *Server) SelfCheckMarkPgStatStatementsOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.PgStatStatements.State = CollectionStateOkay
	s.SelfCheck.PgStatStatements.Msg = "ok"
}

func (s *Server) SelfCheckMarkPgStatStatementsError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.PgStatStatements.State = CollectionStateNotAvailable
	s.SelfCheck.PgStatStatements.Msg = msg
}

// schema information

func (s *Server) SelfCheckMarkSchemaInformationOk(dbName string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SchemaInformation = append(s.SelfCheck.SchemaInformation, DbCollectionState{
		DbName: dbName,
		CollectionState: CollectionState{
			State: CollectionStateOkay,
			Msg:   "ok",
		},
	})
}

func (s *Server) SelfCheckMarkSchemaInformationError(dbName, msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SchemaInformation = append(s.SelfCheck.SchemaInformation, DbCollectionState{
		DbName: dbName,
		CollectionState: CollectionState{
			State: CollectionStateOkay,
			Msg:   msg,
		},
	})
}

// column stats

func (s *Server) SelfCheckMarkColumnStatsOk(dbName string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.ColumnStats = append(s.SelfCheck.ColumnStats, DbCollectionState{
		DbName: dbName,
		CollectionState: CollectionState{
			State: CollectionStateOkay,
			Msg:   "ok",
		},
	})
}

func (s *Server) SelfCheckMarkColumnStatsError(dbName, msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.ColumnStats = append(s.SelfCheck.ColumnStats, DbCollectionState{
		DbName: dbName,
		CollectionState: CollectionState{
			State: CollectionStateOkay,
			Msg:   msg,
		},
	})
}

// extended stats

func (s *Server) SelfCheckMarkExtendedStatsOk(dbName string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.ExtendedStats = append(s.SelfCheck.ExtendedStats, DbCollectionState{
		DbName: dbName,
		CollectionState: CollectionState{
			State: CollectionStateOkay,
			Msg:   "ok",
		},
	})
}

func (s *Server) SelfCheckMarkExtendedStatsError(dbName, msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.ExtendedStats = append(s.SelfCheck.ExtendedStats, DbCollectionState{
		DbName: dbName,
		CollectionState: CollectionState{
			State: CollectionStateOkay,
			Msg:   msg,
		},
	})
}

// Log Insights

func (s *Server) SelfCheckMarkLogInsightsOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.LogInsights.State = CollectionStateOkay
	s.SelfCheck.LogInsights.Msg = "ok"
}

func (s *Server) SelfCheckMarkLogInsightsNotAvailable(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.LogInsights.State = CollectionStateNotAvailable
	s.SelfCheck.LogInsights.Msg = msg
}

func (s *Server) SelfCheckMarkLogInsightsError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.LogInsights.State = CollectionStateNotAvailable
	s.SelfCheck.LogInsights.Msg = msg
}

// Automated EXPLAIN

func (s *Server) SelfCheckMarkAutomatedExplainOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.AutomatedExplain.State = CollectionStateOkay
	s.SelfCheck.AutomatedExplain.Msg = "ok"
}

func (s *Server) SelfCheckMarkAutomatedExplainNotAvailable(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.AutomatedExplain.State = CollectionStateNotAvailable
	s.SelfCheck.AutomatedExplain.Msg = msg
}

func (s *Server) SelfCheckMarkAutomatedExplainError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.AutomatedExplain.State = CollectionStateNotAvailable
	s.SelfCheck.AutomatedExplain.Msg = msg
}
