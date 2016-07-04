package state

// http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.html
type RdsOsSnapshot struct {
	Engine             string  `json:"engine"`             // The database engine for the DB instance.
	InstanceID         string  `json:"instanceID"`         // The DB instance identifier.
	InstanceResourceID string  `json:"instanceResourceID"` // A region-unique, immutable identifier for the DB instance, also used as the log stream identifier.
	Timestamp          string  `json:"timestamp"`          // The time at which the metrics were taken.
	Version            float32 `json:"version"`            // The version of the OS metrics' stream JSON format.
	Uptime             string  `json:"uptime"`             // The amount of time that the DB instance has been active.
	NumVCPUs           int32   `json:"numVCPUs"`           // The number of virtual CPUs for the DB instance.

	CPUUtilization    RdsOsCPUUtilization     `json:"cpuUtilization"`
	LoadAverageMinute RdsOsLoadAverageMinute  `json:"loadAverageMinute"`
	Memory            RdsOsMemory             `json:"memory"`
	Swap              RdsOsSwap               `json:"swap"`
	Network           []RdsOsNetworkInterface `json:"network"`
	DiskIO            []RdsOsDiskIO           `json:"diskIO"`
	FileSystems       []RdsOsFileSystem       `json:"fileSys"`
}

type RdsOsCPUUtilization struct {
	Guest  float32 `json:"guest"`  // The percentage of CPU in use by guest programs.
	Irq    float32 `json:"irq"`    // The percentage of CPU in use by software interrupts.
	System float32 `json:"system"` // The percentage of CPU in use by the kernel.
	Wait   float32 `json:"wait"`   // The percentage of CPU unused while waiting for I/O access.
	Idle   float32 `json:"idle"`   // The percentage of CPU that is idle.
	User   float32 `json:"user"`   // The percentage of CPU in use by user programs.
	Total  float32 `json:"total"`  // The total percentage of the CPU in use. This value excludes the nice value.
	Steal  float32 `json:"steal"`  // The percentage of CPU in use by other virtual machines.
	Nice   float32 `json:"nice"`   // The percentage of CPU in use by programs running at lowest priority.
}

type RdsOsLoadAverageMinute struct {
	Fifteen float32 `json:"fifteen"` // The number of processes requesting CPU time over the last 15 minutes.
	Five    float32 `json:"five"`    // The number of processes requesting CPU time over the last 5 minutes.
	One     float32 `json:"one"`     // The number of processes requesting CPU time over the last minute.
}

type RdsOsMemory struct {
	Writeback      int64 `json:"writeback"`      // The amount of dirty pages in RAM that are still being written to the backing storage, in kilobytes.
	HugePagesFree  int64 `json:"hugePagesFree"`  // The number of free huge pages. Huge pages are a feature of the Linux kernel.
	HugePagesRsvd  int64 `json:"hugePagesRsvd"`  // The number of committed huge pages.
	HugePagesSurp  int64 `json:"hugePagesSurp"`  // The number of available surplus huge pages over the total.
	Cached         int64 `json:"cached"`         // The amount of memory used for caching file system–based I/O.
	HugePagesSize  int64 `json:"hugePagesSize"`  // The size for each huge pages unit, in kilobytes.
	Free           int64 `json:"free"`           // The amount of unassigned memory, in kilobytes.
	HugePagesTotal int64 `json:"hugePagesTotal"` // The total number of huge pages for the system.
	Inactive       int64 `json:"inactive"`       // The amount of least-frequently used memory pages, in kilobytes.
	PageTables     int64 `json:"pageTables"`     // The amount of memory used by page tables, in kilobytes.
	Dirty          int64 `json:"dirty"`          // The amount of memory pages in RAM that have been modified but not written to their related data block in storage, in kilobytes.
	Mapped         int64 `json:"mapped"`         // The total amount of file-system contents that is memory mapped inside a process address space, in kilobytes.
	Active         int64 `json:"active"`         // The amount of assigned memory, in kilobytes.
	Total          int64 `json:"total"`          // The total amount of memory, in kilobytes.
	Slab           int64 `json:"slab"`           // The amount of reusable kernel data structures, in kilobytes.
	Buffers        int64 `json:"buffers"`        // The amount of memory used for buffering I/O requests prior to writing to the storage device, in kilobytes.
}

type RdsOsSwap struct {
	Cached int64 `json:"cached"` // The amount of swap memory, in kilobytes, used as cache memory.
	Total  int64 `json:"total"`  // The total amount of swap memory available, in kilobytes.
	Free   int64 `json:"free"`   // The total amount of swap memory free, in kilobytes.
}

type RdsOsNetworkInterface struct {
	Interface string  `json:"interface"` // The identifier for the network interface being used for the DB instance.
	Rx        float64 `json:"rx"`        // The number of packets received.
	Tx        float64 `json:"tx"`        // The number of packets uploaded.
}

type RdsOsDiskIO struct {
	WriteKbPS   float32 `json:"writeKbPS"`   // The number of kilobytes written per second.
	ReadIOsPS   float32 `json:"readIOsPS"`   // The number of read operations per second.
	Await       float32 `json:"await"`       // The number of milliseconds required to respond to requests, including queue time and service time.
	ReadKbPS    float32 `json:"readKbPS"`    // The number of kilobytes read per second.
	RrqmPS      float32 `json:"rrqmPS"`      // The number of merged read requests queued per second.
	Util        float32 `json:"util"`        // The percentage of CPU time during which requests were issued.
	AvgQueueLen float32 `json:"avgQueueLen"` // The number of requests waiting in the I/O device's queue.
	Tps         float32 `json:"tps"`         // The number of I/O transactions per second.
	ReadKb      float32 `json:"readKb"`      // The total number of kilobytes read.
	Device      string  `json:"device"`      // The identifier of the disk device in use.
	WriteKb     float32 `json:"writeKb"`     // The total number of kilobytes written.
	AvgReqSz    float32 `json:"avgReqSz"`    // The average request size, in kilobytes.
	WrqmPS      float32 `json:"wrqmPS"`      // The number of merged write requests queued per second.
	WriteIOsPS  float32 `json:"writeIOsPS"`  // The number of write operations per second.
}

type RdsOsFileSystem struct {
	Used            int64   `json:"used"`            // The amount of disk space used by files in the file system, in kilobytes.
	Name            string  `json:"name"`            // The name of the file system.
	UsedFiles       int64   `json:"usedFiles"`       // The number of files in the file system.
	UsedFilePercent float32 `json:"usedFilePercent"` // The percentage of available files in use.
	MaxFiles        int64   `json:"maxFiles"`        // The maximum number of files that can be created for the file system.
	MountPoint      string  `json:"mountPoint"`      // The path to the file system.
	Total           int64   `json:"total"`           // The total number of disk space available for the file system, in kilobytes.
	UsedPercent     float32 `json:"usedPercent"`     // The percentage of the file-system disk space in use.
}
