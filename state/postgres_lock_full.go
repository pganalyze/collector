package state

import (
	"github.com/guregu/null"
)

// PostgresLockFull - Information of both blocked or blocking processes
type PostgresLockFull struct {
	BlockedPid      int32       // The blocked process ID
	BlockedMode     null.String // Name of the lock mode desired by the blocked process, e.g., AccessShareLock
	BlockedLockType null.String // Lock type of the blocked process, with the related info like relation oid
	BlockingPid     int32       // The blocking process ID
	BlockingMode    null.String // Name of the lock mode held by the blocking process, e.g., AccessExclusiveLock
}
