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
	CollectionAspectTelemetry CollectionAspect = iota
	CollectionAspectSystemStats
	CollectionAspectMonitoringDbConnection
	CollectionAspectPgVersion
	CollectionAspectApiConnection
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

type SelfTestStatus struct {
	mutex               *sync.Mutex
	CollectionSuspended struct {
		Value bool
		Msg   string
	}
	MonitoredDbs     []string
	AspectStatuses   map[CollectionAspect]*CollectionAspectStatus
	AspectDbStatuses map[DbCollectionAspect](map[string]*CollectionAspectStatus)
}

func MakeSelfTest() (s *SelfTestStatus) {
	return &SelfTestStatus{
		mutex:            &sync.Mutex{},
		AspectStatuses:   make(map[CollectionAspect]*CollectionAspectStatus),
		AspectDbStatuses: make(map[DbCollectionAspect](map[string]*CollectionAspectStatus)),
	}
}

// collection suspended (e.g., if replica)
func (s *SelfTestStatus) MarkCollectionSuspended(format string, args ...any) {
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

func (s *SelfTestStatus) MarkCollectionAspectOk(aspect CollectionAspect) {
	s.MarkCollectionAspect(aspect, CollectionStateOkay, "ok")
}

func (s *SelfTestStatus) MarkCollectionAspectNotAvailable(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateNotAvailable, format, args...)
}

func (s *SelfTestStatus) MarkCollectionAspectWarning(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateWarning, format, args...)
}

func (s *SelfTestStatus) MarkCollectionAspectError(aspect CollectionAspect, format string, args ...any) {
	s.MarkCollectionAspect(aspect, CollectionStateError, format, args...)
}

func (s *SelfTestStatus) MarkCollectionAspect(aspect CollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
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

func (s *SelfTestStatus) HintCollectionAspect(aspect CollectionAspect, format string, args ...any) {
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

func (s *SelfTestStatus) GetCollectionAspectStatus(aspect CollectionAspect) *CollectionAspectStatus {
	return s.AspectStatuses[aspect]
}

func (s *SelfTestStatus) MarkMonitoredDb(dbName string) {
	if s == nil {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.MonitoredDbs = append(s.MonitoredDbs, dbName)
}

func (s *SelfTestStatus) MarkDbCollectionAspectOk(dbName string, aspect DbCollectionAspect) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateOkay, "ok")
}

func (s *SelfTestStatus) MarkDbCollectionAspectNotAvailable(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateNotAvailable, format, args...)
}

func (s *SelfTestStatus) MarkDbCollectionAspectWarning(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateWarning, format, args...)
}

func (s *SelfTestStatus) MarkDbCollectionAspectError(dbName string, aspect DbCollectionAspect, format string, args ...any) {
	s.MarkDbCollectionAspect(dbName, aspect, CollectionStateError, format, args...)
}

func (s *SelfTestStatus) MarkDbCollectionAspect(dbName string, aspect DbCollectionAspect, state CollectionStateCode, format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
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

func (s *SelfTestStatus) MarkRemainingDbCollectionAspectError(aspect DbCollectionAspect, format string, args ...any) {
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

func (s *SelfTestStatus) HintDbCollectionAspect(dbName string, aspect DbCollectionAspect, state CollectionStateCode, format string, args ...any) {
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

func (s *SelfTestStatus) GetCollectionAspectDbStatus(dbName string, aspect DbCollectionAspect) *CollectionAspectStatus {
	aspectDbStates, ok := s.AspectDbStatuses[aspect]
	if !ok {
		return &CollectionAspectStatus{}
	}

	return aspectDbStates[dbName]
}
