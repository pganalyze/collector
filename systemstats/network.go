package systemstats

// Network - Information about the network activity going in and out of the database
type Network struct {
	ReceiveThroughput  *int64 `json:"receive_throughput"`
	TransmitThroughput *int64 `json:"transmit_throughput"`
}
