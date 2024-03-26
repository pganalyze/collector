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

type SelfCheckStatus struct {
	enabled             bool
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
	Logs                   CollectionState
	AutomatedExplain       CollectionState
}

func (s *Server) SelfCheckInit() {
	s.SelfCheck.enabled = true
}

func (s *Server) SelfCheckMarkMonitoredDb(dbName string) {
	if !s.SelfCheck.enabled {
		return
	}
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SchemaInformation = append(s.SelfCheck.SchemaInformation, DbCollectionState{
		DbName: dbName,
	})
	s.SelfCheck.ColumnStats = append(s.SelfCheck.ColumnStats, DbCollectionState{
		DbName: dbName,
	})
	s.SelfCheck.ExtendedStats = append(s.SelfCheck.ExtendedStats, DbCollectionState{
		DbName: dbName,
	})
}

// collection suspended (e.g., if replica)
func (s *Server) SelfCheckMarkCollectionSuspended(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.CollectionSuspended.Value = true
	s.SelfCheck.CollectionSuspended.Msg = msg
}

// collector stats
func (s *Server) SelfCheckMarkCollectorTelemetryOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.CollectorTelemetry.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.CollectorTelemetry.State = CollectionStateOkay
	s.SelfCheck.CollectorTelemetry.Msg = "ok"
}

func (s *Server) SelfCheckMarkCollectorTelemetryError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.CollectorTelemetry.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.CollectorTelemetry.State = CollectionStateError
	s.SelfCheck.CollectorTelemetry.Msg = msg
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

func (s *Server) SelfCheckMarkSystemStatsNotAvailable(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SystemStats.State = CollectionStateNotAvailable
	s.SelfCheck.SystemStats.Msg = msg
}

func (s *Server) SelfCheckMarkSystemStatsError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.SystemStats.State = CollectionStateError
	// note: here we can hit errors and proceed in some cases; collect all errors
	if s.SelfCheck.SystemStats.Msg != "" {
		s.SelfCheck.SystemStats.Msg += "; "
	}
	s.SelfCheck.SystemStats.Msg += msg
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
	if s.SelfCheck.PgStatStatements.State != CollectionStateUnchecked {
		return
	}
	s.SelfCheck.PgStatStatements.State = CollectionStateOkay
	s.SelfCheck.PgStatStatements.Msg = "ok"
}

func (s *Server) SelfCheckMarkPgStatStatementsWarning(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.PgStatStatements.State = CollectionStateWarning
	s.SelfCheck.PgStatStatements.Msg = msg
}

func (s *Server) SelfCheckMarkPgStatStatementsError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.PgStatStatements.State = CollectionStateError
	s.SelfCheck.PgStatStatements.Msg = msg
}

// schema information
func (s *Server) SelfCheckMarkSchemaOk(dbName string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.SchemaInformation {
		if info.DbName == dbName {
			s.SelfCheck.SchemaInformation[i].State = CollectionStateOkay
			s.SelfCheck.SchemaInformation[i].Msg = "ok"
			return
		}
	}
}

func (s *Server) SelfCheckMarkSchemaError(dbName, msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.SchemaInformation {
		if info.DbName == dbName {
			s.SelfCheck.SchemaInformation[i].State = CollectionStateError
			s.SelfCheck.SchemaInformation[i].Msg = msg
			return
		}
	}
}

func (s *Server) SelfCheckMarkAllRemainingSchemaError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.SchemaInformation {
		if info.State == CollectionStateUnchecked {
			s.SelfCheck.SchemaInformation[i].State = CollectionStateError
			s.SelfCheck.SchemaInformation[i].Msg = msg
			return
		}
	}
}

// column stats
func (s *Server) SelfCheckMarkColumnStatsOk(dbName string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.ColumnStats {
		if info.DbName == dbName {
			s.SelfCheck.ColumnStats[i].State = CollectionStateOkay
			s.SelfCheck.ColumnStats[i].Msg = "ok"
			return
		}
	}
}

func (s *Server) SelfCheckMarkColumnStatsError(dbName, msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.ColumnStats {
		if info.DbName == dbName {
			s.SelfCheck.ColumnStats[i].State = CollectionStateError
			s.SelfCheck.ColumnStats[i].Msg = msg
			return
		}
	}
}

// extended stats
func (s *Server) SelfCheckMarkExtendedStatsOk(dbName string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.ExtendedStats {
		if info.DbName == dbName {
			s.SelfCheck.ExtendedStats[i].State = CollectionStateOkay
			s.SelfCheck.ExtendedStats[i].Msg = "ok"
			return
		}
	}
}

func (s *Server) SelfCheckMarkExtendedStatsError(dbName, msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	for i, info := range s.SelfCheck.ExtendedStats {
		if info.DbName == dbName {
			s.SelfCheck.ExtendedStats[i].State = CollectionStateError
			s.SelfCheck.ExtendedStats[i].Msg = msg
			return
		}
	}
}

// Log Insights
func (s *Server) SelfCheckMarkLogsOk() {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	if s.SelfCheck.Logs.State != CollectionStateUnchecked {
		return
	}

	s.SelfCheck.Logs.State = CollectionStateOkay
	s.SelfCheck.Logs.Msg = "ok; available in 5-10m"
}

func (s *Server) SelfCheckMarkLogsNotAvailable(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.Logs.State = CollectionStateNotAvailable
	s.SelfCheck.Logs.Msg = msg
}

func (s *Server) SelfCheckMarkLogsError(msg string) {
	s.selfCheckMutex.Lock()
	defer s.selfCheckMutex.Unlock()
	s.SelfCheck.Logs.State = CollectionStateError
	s.SelfCheck.Logs.Msg = msg
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
	s.SelfCheck.AutomatedExplain.State = CollectionStateError
	s.SelfCheck.AutomatedExplain.Msg = msg
}
