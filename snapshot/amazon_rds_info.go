//go:generate msgp

package snapshot

// AmazonRdsInfo - Additional information for Amazon RDS systems

// http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.html

type RdsOsNetworkInterface struct {
	Interface string  `msg:"interface"` // The identifier for the network interface being used for the DB instance.
	Rx        float64 `msg:"rx"`        // The number of packets received.
	Tx        float64 `msg:"tx"`        // The number of packets uploaded.
}

type RdsOsFileSystem struct {
	Used            int64   `msg:"used"`            // The amount of disk space used by files in the file system, in kilobytes.
	Name            string  `msg:"name"`            // The name of the file system.
	UsedFiles       int64   `msg:"usedFiles"`       // The number of files in the file system.
	UsedFilePercent float32 `msg:"usedFilePercent"` // The percentage of available files in use.
	MaxFiles        int64   `msg:"maxFiles"`        // The maximum number of files that can be created for the file system.
	MountPoint      string  `msg:"mountPoint"`      // The path to the file system.
	Total           int64   `msg:"total"`           // The total number of disk space available for the file system, in kilobytes.
	UsedPercent     float32 `msg:"usedPercent"`     // The percentage of the file-system disk space in use.
}

type RdsOsProcess struct {
	Vss          int64   `msg:"vss"`          // The amount of virtual memory allocated to the process, in kilobytes.
	Name         string  `msg:"name"`         // The name of the process.
	Tgid         int64   `msg:"tgid"`         // The thread group identifier, which is a number representing the process ID to which a thread belongs. This identifier is used to group threads from the same process.
	ParentID     int64   `msg:"parentID"`     // The process identifier for the parent process of the process.
	MemoryUsedPc float32 `msg:"memoryUsedPc"` // The percentage of memory used by the process.
	CPUUsedPc    float32 `msg:"cpuUsedPc"`    // The percentage of CPU used by the process.
	ID           int64   `msg:"id"`           // The identifier of the process.
	Rss          int64   `msg:"rss"`          // The amount of RAM allocated to the process, in kilobytes.
}
