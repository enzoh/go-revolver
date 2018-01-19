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
	// It also accepts a list of excluded peers, which won't be included in the
	// recommended list.  The idea is that the caller may not want to gossip to
	// these peers because they might already have the artifact.
	Recommend(count int, exclude []peer.ID) []peer.ID

	// Size returns the number of peers in the routing table.
	Size() int

	// Shutdown cleans up any resources that the RoutingTable might've allocated
	Shutdown()
}
