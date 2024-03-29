package state

import (
	"fmt"
	"sync"
)

// What do users care about?
//  - does feature X work? If not, what can I do to fix it?
// How can we tell them that?
//  - identify whether all the subsystems necessary for a feature to work are working correctly
//  - check whether errors match common errors with known causes and easy-to-communicate fixes
//  - summarize the information about subsystems in terms of the features that depend on them

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

type CollectionAspect int

const (
	CollectionAspectTelemetry CollectionAspect = iota
	CollectionAspectSystemStats
	CollectionAspectMonitoringDbConnection
	CollectionAspectPgStatStatements
	CollectionAspectLogs
	CollectionAspectExplain
)

type DbCollectionAspect int

const (
	CollectionAspectSchemaInformation DbCollectionAspect = iota
	CollectionAspectColumnStats
	CollectionAspectExtendedStats
)

type SelfCheckStatus struct {
	mutex               *sync.Mutex
	CollectionSuspended struct {
		Value bool
		Msg   string
	}
	MonitoredDbs     []string
	AspectStatuses   map[CollectionAspect]*CollectionAspectStatus
	AspectDbStatuses map[DbCollectionAspect](map[string]*CollectionAspectStatus)
}

func MakeSelfCheck() (s *SelfCheckStatus) {
	return &SelfCheckStatus{
		mutex:            &sync.Mutex{},
		AspectStatuses:   make(map[CollectionAspect]*CollectionAspectStatus),
		AspectDbStatuses: make(map[DbCollectionAspect](map[string]*CollectionAspectStatus)),
	}
}

// collection suspended (e.g., if replica)
func (s *SelfCheckStatus) MarkCollectionSuspended(format string, args ...any) {
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

func (s *SelfCheckStatus) MarkCollectionAspectOk(aspect CollectionAspect) {
	s.MarkCollectionAspect(aspect, CollectionStateOkay, "ok")
}

func (s *SelfCheckStatus) MarkCollectionAspectNotAvailable(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateNotAvailable, format, args...)
}

func (s *SelfCheckStatus) MarkCollectionAspectWarning(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateWarning, format, args...)
}

func (s *SelfCheckStatus) MarkCollectionAspectError(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateError, format, args...)
}

func (s *SelfCheckStatus) MarkCollectionAspect(aspect CollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format+"\n", args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectState, ok := s.AspectStatuses[aspect]
	if !ok {
		aspectState = &CollectionAspectStatus{}
		s.AspectStatuses[aspect] = aspectState
	}
	if skipUpdate(aspectState.State, state) {
		return
	}
	aspectState.State = state
	aspectState.Msg = msg
}

func (s *SelfCheckStatus) HintCollectionAspect(aspect CollectionAspect, format string, args ...any) {
	if s == nil {
		return
	}
	hint := fmt.Sprintf(format+"\n", args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectState, ok := s.AspectStatuses[aspect]
	if !ok {
		aspectState = &CollectionAspectStatus{}
		s.AspectStatuses[aspect] = aspectState
	}
	if skipHintUpdate(aspectState.Hint, hint) {
		return
	}
	aspectState.Hint = hint
}

func (s *SelfCheckStatus) GetCollectionAspectStatus(aspect CollectionAspect) *CollectionAspectStatus {
	return s.AspectStatuses[aspect]
}

func (s *SelfCheckStatus) MarkMonitoredDb(dbName string) {
	if s == nil {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.MonitoredDbs = append(s.MonitoredDbs, dbName)
}

func (s *SelfCheckStatus) MarkDbCollectionAspectOk(dbName string, aspect DbCollectionAspect) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateOkay, "ok")
}

func (s *SelfCheckStatus) MarkDbCollectionAspectNotAvailable(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateNotAvailable, format, args...)
}

func (s *SelfCheckStatus) MarkDbCollectionAspectWarning(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateWarning, format, args...)
}

func (s *SelfCheckStatus) MarkDbCollectionAspectError(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateError, format, args...)
}

func (s *SelfCheckStatus) MarkDbCollectionAspect(dbName string, aspect DbCollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format+"\n", args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectDbStates, ok := s.AspectDbStatuses[aspect]
	if !ok {
		aspectDbStates = make(map[string]*CollectionAspectStatus)
		s.AspectDbStatuses[aspect] = aspectDbStates
	}

	aspectDbState, ok := aspectDbStates[dbName]
	if !ok {
		aspectDbState = &CollectionAspectStatus{}
		aspectDbStates[dbName] = aspectDbState
	}

	if skipUpdate(aspectDbState.State, state) {
		return
	}

	aspectDbState.State = state
	aspectDbState.Msg = msg
}

func (s *SelfCheckStatus) MarkRemainingDbCollectionAspectError(aspect DbCollectionAspect, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format+"\n", args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectDbStates, ok := s.AspectDbStatuses[aspect]
	if !ok {
		// nothing to do
		return
	}

	for _, aspectDbState := range aspectDbStates {
		if skipUpdate(aspectDbState.State, CollectionStateError) {
			continue
		}
		aspectDbState.State = CollectionStateError
		aspectDbState.Msg = msg
	}
}

func (s *SelfCheckStatus) HintDbCollectionAspect(dbName string, aspect DbCollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	hint := fmt.Sprintf(format+"\n", args...)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	aspectDbStates, ok := s.AspectDbStatuses[aspect]
	if !ok {
		aspectDbStates = make(map[string]*CollectionAspectStatus)
		s.AspectDbStatuses[aspect] = aspectDbStates
	}

	aspectDbState, ok := aspectDbStates[dbName]
	if !ok {
		aspectDbState = &CollectionAspectStatus{}
		aspectDbStates[dbName] = aspectDbState
	}

	if skipHintUpdate(aspectDbState.Hint, hint) {
		return
	}

	aspectDbState.Hint = hint
}

func (s *SelfCheckStatus) GetCollectionAspectDbStatus(dbName string, aspect DbCollectionAspect) *CollectionAspectStatus {
	aspectDbStates, ok := s.AspectDbStatuses[aspect]
	if !ok {
		return &CollectionAspectStatus{}
	}

	return aspectDbStates[dbName]
}
