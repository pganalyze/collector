package state

// PostgresLock - Information of both blocked or blocking processes
type PostgresLock struct {
	BlockedPid   int32   // The blocked process ID
	BlockingPids []int64 // The list of blocking process ID
}
