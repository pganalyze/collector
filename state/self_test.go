package state

type CollectionStateCode int

const (
	CollectionStateUnchecked CollectionStateCode = iota
	CollectionStateNotAvailable
	CollectionStateWarning
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
	State  CollectionStateCode
	Msg    string
	DbName string
}

type SelfTestStatus struct {
	CollectionSuspended struct {
		Value bool
		Msg   string
	}
	CollectorTelemetry     CollectionState
	SystemStats            CollectionState
	MonitoringDbConnection CollectionState
	PgStatStatements       CollectionState
	SchemaInformation      []DbCollectionState
	ColumnStats            []DbCollectionState
	ExtendedStats          []DbCollectionState
	LogInsights            CollectionState
	AutomatedExplain       CollectionState
}

func (s *Server) SelfTestMarkMonitoredDb(dbName string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.SchemaInformation = append(s.SelfTest.SchemaInformation, DbCollectionState{
		DbName: dbName,
	})
	s.SelfTest.ColumnStats = append(s.SelfTest.ColumnStats, DbCollectionState{
		DbName: dbName,
	})
	s.SelfTest.ExtendedStats = append(s.SelfTest.ExtendedStats, DbCollectionState{
		DbName: dbName,
	})
}

// collection suspended (e.g., if replica)
func (s *Server) SelfTestMarkCollectionSuspended(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.CollectionSuspended.Value = true
	s.SelfTest.CollectionSuspended.Msg = msg
}

// collector stats
func (s *Server) SelfTestMarkCollectorTelemetryOk() {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	if s.SelfTest.CollectorTelemetry.State != CollectionStateUnchecked {
		return
	}
	s.SelfTest.CollectorTelemetry.State = CollectionStateOkay
	s.SelfTest.CollectorTelemetry.Msg = "ok"
}

func (s *Server) SelfTestMarkCollectorTelemetryError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	if s.SelfTest.CollectorTelemetry.State != CollectionStateUnchecked {
		return
	}
	s.SelfTest.CollectorTelemetry.State = CollectionStateError
	s.SelfTest.CollectorTelemetry.Msg = msg
}

// system stats
func (s *Server) SelfTestMarkSystemStatsOk() {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	if s.SelfTest.SystemStats.State != CollectionStateUnchecked {
		return
	}
	s.SelfTest.SystemStats.State = CollectionStateOkay
	s.SelfTest.SystemStats.Msg = "ok"
}

func (s *Server) SelfTestMarkSystemStatsNotAvailable(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.SystemStats.State = CollectionStateNotAvailable
	s.SelfTest.SystemStats.Msg = msg
}

func (s *Server) SelfTestMarkSystemStatsError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.SystemStats.State = CollectionStateError
	// note: here we can hit errors and proceed in some cases; collect all errors
	if s.SelfTest.SystemStats.Msg != "" {
		s.SelfTest.SystemStats.Msg += "; "
	}
	s.SelfTest.SystemStats.Msg += msg
}

// monitoring DB connection
func (s *Server) SelfTestMarkMonitoringDbConnectionOk() {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.MonitoringDbConnection.State = CollectionStateOkay
	s.SelfTest.MonitoringDbConnection.Msg = "ok"
}

func (s *Server) SelfTestMarkMonitoringDbConnectionError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.MonitoringDbConnection.State = CollectionStateNotAvailable
	s.SelfTest.MonitoringDbConnection.Msg = msg
}

// pg_stat_statements
func (s *Server) SelfTestMarkPgStatStatementsOk() {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	if s.SelfTest.PgStatStatements.State != CollectionStateUnchecked {
		return
	}
	s.SelfTest.PgStatStatements.State = CollectionStateOkay
	s.SelfTest.PgStatStatements.Msg = "ok"
}

func (s *Server) SelfTestMarkPgStatStatementsWarning(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.PgStatStatements.State = CollectionStateWarning
	s.SelfTest.PgStatStatements.Msg = msg
}

func (s *Server) SelfTestMarkPgStatStatementsError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.PgStatStatements.State = CollectionStateError
	s.SelfTest.PgStatStatements.Msg = msg
}

// schema information
func (s *Server) SelfTestMarkSchemaOk(dbName string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.SchemaInformation {
		if info.DbName == dbName {
			s.SelfTest.SchemaInformation[i].State = CollectionStateOkay
			s.SelfTest.SchemaInformation[i].Msg = "ok"
			return
		}
	}
}

func (s *Server) SelfTestMarkSchemaError(dbName, msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.SchemaInformation {
		if info.DbName == dbName {
			s.SelfTest.SchemaInformation[i].State = CollectionStateError
			s.SelfTest.SchemaInformation[i].Msg = msg
			return
		}
	}
}

func (s *Server) SelfTestMarkAllRemainingSchemaError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.SchemaInformation {
		if info.State == CollectionStateUnchecked {
			s.SelfTest.SchemaInformation[i].State = CollectionStateError
			s.SelfTest.SchemaInformation[i].Msg = msg
			return
		}
	}
}

// column stats
func (s *Server) SelfTestMarkColumnStatsOk(dbName string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.ColumnStats {
		if info.DbName == dbName {
			s.SelfTest.ColumnStats[i].State = CollectionStateOkay
			s.SelfTest.ColumnStats[i].Msg = "ok"
			return
		}
	}
}

func (s *Server) SelfTestMarkColumnStatsError(dbName, msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.ColumnStats {
		if info.DbName == dbName {
			s.SelfTest.ColumnStats[i].State = CollectionStateError
			s.SelfTest.ColumnStats[i].Msg = msg
			return
		}
	}
}

// extended stats
func (s *Server) SelfTestMarkExtendedStatsOk(dbName string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.ExtendedStats {
		if info.DbName == dbName {
			s.SelfTest.ExtendedStats[i].State = CollectionStateOkay
			s.SelfTest.ExtendedStats[i].Msg = "ok"
			return
		}
	}
}

func (s *Server) SelfTestMarkExtendedStatsError(dbName, msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	for i, info := range s.SelfTest.ExtendedStats {
		if info.DbName == dbName {
			s.SelfTest.ExtendedStats[i].State = CollectionStateError
			s.SelfTest.ExtendedStats[i].Msg = msg
			return
		}
	}
}

// Log Insights
func (s *Server) SelfTestMarkLogInsightsOk() {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.LogInsights.State = CollectionStateOkay
	s.SelfTest.LogInsights.Msg = "ok"
}

func (s *Server) SelfTestMarkLogInsightsNotAvailable(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.LogInsights.State = CollectionStateNotAvailable
	s.SelfTest.LogInsights.Msg = msg
}

func (s *Server) SelfTestMarkLogInsightsError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.LogInsights.State = CollectionStateNotAvailable
	s.SelfTest.LogInsights.Msg = msg
}

// Automated EXPLAIN
func (s *Server) SelfTestMarkAutomatedExplainOk() {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.AutomatedExplain.State = CollectionStateOkay
	s.SelfTest.AutomatedExplain.Msg = "ok"
}

func (s *Server) SelfTestMarkAutomatedExplainNotAvailable(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.AutomatedExplain.State = CollectionStateNotAvailable
	s.SelfTest.AutomatedExplain.Msg = msg
}

func (s *Server) SelfTestMarkAutomatedExplainError(msg string) {
	s.selfTestMutex.Lock()
	defer s.selfTestMutex.Unlock()
	s.SelfTest.AutomatedExplain.State = CollectionStateNotAvailable
	s.SelfTest.AutomatedExplain.Msg = msg
}
