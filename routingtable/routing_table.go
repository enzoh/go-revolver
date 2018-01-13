package routingtable

import (
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// RoutingTable stores peers and recommends peers for gossiping purposes.
type RoutingTable interface {
	// Add a peer to a routing table.
	Add(pid peer.ID)

	// Remove a peer from a routing table.
	Remove(pid peer.ID)

	// Recommend a slice of peers from a routing table for gossiping purposes.
	// It also accepts a list of "preferred" peers.  The idea is that the caller
	// would prefer peers that are currently connected, so we try to include the
	// preferred peers in the recommended list if possible.
	Recommend(count int, preferred []peer.ID) []peer.ID

	// Sample returns a subset of peers in the routing table.
	Sample() []peer.ID

	// Size returns the number of peers in the routing table.
	Size() int

	// Shutdown cleans up any resources that the RoutingTable might've allocated
	Shutdown()
}
