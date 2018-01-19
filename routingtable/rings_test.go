package routingtable

import (
	"strconv"
	"testing"
	"time"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// A latency probe function that always returns a constant.
func fixedLatencyProbe(_ peer.ID) (time.Duration, error) {
	return 256 * time.Millisecond, nil
}

// Return a list of `n` unique peer IDs
func uniquePIDs(n int) []peer.ID {
	var pids []peer.ID
	for i := 0; i < n; i++ {
		pids = append(pids, peer.ID(strconv.Itoa(i)))
	}
	return pids
}

// Test that given a sufficiently populated routing table, `Recommend` always
// returns the requested number of nodes.
func TestRecommendEnoughNodes(t *testing.T) {
	// We use a probe function that returns a fixed latency.  The idea is that
	// since the latency is fixed, all nodes will be put into the same ring,
	// which means the other rings will be empty.  Thus we can see if
	// `Recommend` returns the requested number of nodes even if some rings are
	// empty.
	table := NewRingsRoutingTable(NewDefaultRingsConfig(fixedLatencyProbe))

	for _, pid := range uniquePIDs(100) {
		table.Add(pid)
	}

	recommended := table.Recommend(10, nil)
	if len(recommended) != 10 {
		t.Fatalf("Should recommend 10 peers, but actually recommended %v peers.",
			len(recommended))
	}
}
