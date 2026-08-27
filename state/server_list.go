package state

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pganalyze/collector/config"
)

// How long the state of servers removed from monitoring at runtime is kept
// around, in case the same server (e.g. an Aurora reader with the same
// identifier) is added back
const retainedStateTTL = 14 * 24 * time.Hour

// ServerList holds the currently monitored servers. The slice is immutable -
// any change swaps in a new slice - so callers can iterate a consistent
// snapshot without locking. The set of servers only changes at runtime when
// server discovery (e.g. Aurora cluster members) is in use; otherwise it is
// set once at startup.
type ServerList struct {
	servers atomic.Pointer[[]*Server]

	// State of servers that were removed from monitoring at runtime, kept so
	// it can be persisted to the state file and recovered if the same server
	// is added back (e.g. an auto-scaling reader cycling off and on)
	retainedMutex sync.Mutex
	retained      map[config.ServerIdentifier]retainedServerState
}

type retainedServerState struct {
	PrevState         PersistedState
	HighFreqPrevState PersistedHighFreqState
	RetainedAt        time.Time
}

func NewServerList() *ServerList {
	list := &ServerList{
		retained: make(map[config.ServerIdentifier]retainedServerState),
	}
	list.servers.Store(&[]*Server{})
	return list
}

// Load returns the current set of monitored servers. The returned slice must
// not be modified.
func (list *ServerList) Load() []*Server {
	return *list.servers.Load()
}

// Store replaces the current set of monitored servers.
func (list *ServerList) Store(servers []*Server) {
	list.servers.Store(&servers)
}

// Retain remembers the state of a server that is being removed from
// monitoring, so it can be written to the state file and recovered in case
// the server comes back.
func (list *ServerList) Retain(server *Server) {
	server.StateMutex.Lock()
	prevState := server.PrevState
	server.StateMutex.Unlock()
	server.HighFreqStateMutex.Lock()
	highFreqPrevState := server.HighFreqPrevState
	server.HighFreqStateMutex.Unlock()

	list.retainedMutex.Lock()
	defer list.retainedMutex.Unlock()
	for identifier, retained := range list.retained {
		if time.Since(retained.RetainedAt) > retainedStateTTL {
			delete(list.retained, identifier)
		}
	}
	list.retained[server.Config.Identifier] = retainedServerState{
		PrevState:         prevState,
		HighFreqPrevState: highFreqPrevState,
		RetainedAt:        time.Now(),
	}
}

// TakeRetained recovers the retained state for a server that is added back to
// monitoring, removing it from the retained set.
func (list *ServerList) TakeRetained(identifier config.ServerIdentifier) (PersistedState, PersistedHighFreqState, bool) {
	list.retainedMutex.Lock()
	defer list.retainedMutex.Unlock()
	retained, ok := list.retained[identifier]
	if !ok {
		return PersistedState{}, PersistedHighFreqState{}, false
	}
	delete(list.retained, identifier)
	return retained.PrevState, retained.HighFreqPrevState, true
}

func (list *ServerList) retainedStates() map[config.ServerIdentifier]retainedServerState {
	list.retainedMutex.Lock()
	defer list.retainedMutex.Unlock()
	retained := make(map[config.ServerIdentifier]retainedServerState, len(list.retained))
	for identifier, state := range list.retained {
		retained[identifier] = state
	}
	return retained
}
