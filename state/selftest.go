package state

import (
	"fmt"
	"sync"
)

// The self-test mechanism is intended to catch errors and potential problems
// encountered during a test surface them to users in a test summary, and
// communicate how these errors will impact the various features of pganalyze.

type CollectionStateCode int

const (
	CollectionStateUnchecked CollectionStateCode = iota
	CollectionStateNotAvailable
	CollectionStateWarning
	CollectionStateError
	CollectionStateOkay
)

type CollectionAspectStatus struct {
	State CollectionStateCode
	Msg   string
	Hint  string
}

type CollectionAspect int

const (
	CollectionAspectApiConnection CollectionAspect = iota
	CollectionAspectWebSocket
	CollectionAspectTelemetry
	CollectionAspectSystemStats
	CollectionAspectMonitoringDbConnection
	CollectionAspectPgVersion
	CollectionAspectPgStatStatements
	CollectionAspectActivity
	CollectionAspectLogs
	CollectionAspectExplain
)

var CollectionAspects = []CollectionAspect{
	CollectionAspectApiConnection,
	CollectionAspectWebSocket,
	CollectionAspectTelemetry,
	CollectionAspectSystemStats,
	CollectionAspectMonitoringDbConnection,
	CollectionAspectPgVersion,
	CollectionAspectPgStatStatements,
	CollectionAspectActivity,
	CollectionAspectLogs,
	CollectionAspectExplain,
}

type DbCollectionAspect int

const (
	CollectionAspectSchema DbCollectionAspect = iota
	CollectionAspectColumnStats
	CollectionAspectExtendedStats
)

var DbCollectionAspects = []DbCollectionAspect{
	CollectionAspectSchema,
	CollectionAspectColumnStats,
	CollectionAspectExtendedStats,
}

type SelfTestResult struct {
	mutex               *sync.Mutex
	CollectionSuspended struct {
		Value bool
		Msg   string
	}
	MonitoredDbs        []string
	AspectStatuses      map[CollectionAspect]*CollectionAspectStatus
	AllDbAspectStatuses map[DbCollectionAspect](map[string]*CollectionAspectStatus)
}

func MakeSelfTest() (s *SelfTestResult) {
	return &SelfTestResult{
		mutex:               &sync.Mutex{},
		AspectStatuses:      make(map[CollectionAspect]*CollectionAspectStatus),
		AllDbAspectStatuses: make(map[DbCollectionAspect](map[string]*CollectionAspectStatus)),
	}
}

// collection suspended (e.g., if replica)
func (s *SelfTestResult) MarkCollectionSuspended(format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format+"\n", args...)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CollectionSuspended.Value = true
	s.CollectionSuspended.Msg = msg
}

func skipUpdate(recordedState, incomingState CollectionStateCode) bool {
	// Always stick with the first error, unless we're overriding an okay state
	// with another state (we may have marked things okay prematurely), or we're
	// escalating a warning to an error
	doUpdate := (recordedState == CollectionStateUnchecked) ||
		(recordedState == CollectionStateOkay && incomingState != CollectionStateOkay) ||
		(recordedState == CollectionStateWarning && incomingState == CollectionStateError)
	return !doUpdate
}

func skipHintUpdate(recordedHint, _incomingHint string) bool {
	// If a hint is already set, don't override it; this matches the behavior
	// above (okay states can be overriden, but they should not have hints
	// because there's nothing to hint about)
	return recordedHint != ""
}

func (s *SelfTestResult) MarkCollectionAspectOk(aspect CollectionAspect) {
	s.MarkCollectionAspect(aspect, CollectionStateOkay, "ok")
}

func (s *SelfTestResult) MarkCollectionAspectNotAvailable(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateNotAvailable, format, args...)
}

func (s *SelfTestResult) MarkCollectionAspectWarning(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateWarning, format, args...)
}

func (s *SelfTestResult) MarkCollectionAspectError(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateError, format, args...)
}

func (s *SelfTestResult) MarkCollectionAspect(aspect CollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectStatus, ok := s.AspectStatuses[aspect]
	if !ok {
		aspectStatus = &CollectionAspectStatus{}
		s.AspectStatuses[aspect] = aspectStatus
	}
	if skipUpdate(aspectStatus.State, state) {
		return
	}
	aspectStatus.State = state
	aspectStatus.Msg = msg
}

func (s *SelfTestResult) HintCollectionAspect(aspect CollectionAspect, format string, args ...any) {
	if s == nil {
		return
	}
	hint := fmt.Sprintf(format, args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectStatus, ok := s.AspectStatuses[aspect]
	if !ok {
		aspectStatus = &CollectionAspectStatus{}
		s.AspectStatuses[aspect] = aspectStatus
	}
	if skipHintUpdate(aspectStatus.Hint, hint) {
		return
	}
	aspectStatus.Hint = hint
}

func (s *SelfTestResult) GetCollectionAspectStatus(aspect CollectionAspect) *CollectionAspectStatus {
	return s.AspectStatuses[aspect]
}

func (s *SelfTestResult) MarkMonitoredDb(dbName string) {
	if s == nil {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.MonitoredDbs = append(s.MonitoredDbs, dbName)
}

func (s *SelfTestResult) MarkDbCollectionAspectOk(dbName string, aspect DbCollectionAspect) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateOkay, "ok")
}

func (s *SelfTestResult) MarkDbCollectionAspectNotAvailable(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateNotAvailable, format, args...)
}

func (s *SelfTestResult) MarkDbCollectionAspectWarning(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateWarning, format, args...)
}

func (s *SelfTestResult) MarkDbCollectionAspectError(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateError, format, args...)
}

func (s *SelfTestResult) MarkDbCollectionAspect(dbName string, aspect DbCollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	dbAspectStatuses, ok := s.AllDbAspectStatuses[aspect]
	if !ok {
		dbAspectStatuses = make(map[string]*CollectionAspectStatus)
		s.AllDbAspectStatuses[aspect] = dbAspectStatuses
	}

	dbAspectStatus, ok := dbAspectStatuses[dbName]
	if !ok {
		dbAspectStatus = &CollectionAspectStatus{}
		dbAspectStatuses[dbName] = dbAspectStatus
	}

	if skipUpdate(dbAspectStatus.State, state) {
		return
	}

	dbAspectStatus.State = state
	dbAspectStatus.Msg = msg
}

func (s *SelfTestResult) MarkRemainingDbCollectionAspectError(aspect DbCollectionAspect, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format+"\n", args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	dbAspectStatuses, ok := s.AllDbAspectStatuses[aspect]
	if !ok {
		// nothing to do
		return
	}

	for _, dbAspectStatus := range dbAspectStatuses {
		if skipUpdate(dbAspectStatus.State, CollectionStateError) {
			continue
		}
		dbAspectStatus.State = CollectionStateError
		dbAspectStatus.Msg = msg
	}
}

func (s *SelfTestResult) HintDbCollectionAspect(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	if s == nil {
		return
	}
	hint := fmt.Sprintf(format, args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectDbStatuses, ok := s.AllDbAspectStatuses[aspect]
	if !ok {
		aspectDbStatuses = make(map[string]*CollectionAspectStatus)
		s.AllDbAspectStatuses[aspect] = aspectDbStatuses
	}

	aspectDbStatus, ok := aspectDbStatuses[dbName]
	if !ok {
		aspectDbStatus = &CollectionAspectStatus{}
		aspectDbStatuses[dbName] = aspectDbStatus
	}

	if skipHintUpdate(aspectDbStatus.Hint, hint) {
		return
	}

	aspectDbStatus.Hint = hint
}

func (s *SelfTestResult) GetDbCollectionAspectStatus(dbName string, aspect DbCollectionAspect) *CollectionAspectStatus {
	aspectDbStatuses, ok := s.AllDbAspectStatuses[aspect]
	if !ok {
		return &CollectionAspectStatus{}
	}

	return aspectDbStatuses[dbName]
}
