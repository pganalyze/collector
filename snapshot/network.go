//go:generate msgp

package snapshot

// Network - Information about the network activity going in and out of the database
type Network struct {
	ReceiveThroughput  NullableInt `msg:"receive_throughput"`
	TransmitThroughput NullableInt `msg:"transmit_throughput"`
}
