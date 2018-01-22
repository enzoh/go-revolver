package routingtable

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// A latency probe function that always returns a constant.
func fixedLatencyProbe(_ peer.ID) (time.Duration, error) {
	return 256 * time.Millisecond, nil
}

// A latency probe that returns random latencies.
func randomLatencyProbe(_ peer.ID) (time.Duration, error) {
	return time.Duration(rand.Intn(1000)) * time.Millisecond, nil
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

// Test that the routing table recommends nodes of varying distances
func TestRecommendVaryingDistances(t *testing.T) {
	// Generate random latency
	pids := uniquePIDs(1000)
	latencyTable := make(map[peer.ID]time.Duration)
	for _, pid := range pids {
		latencyTable[pid] = time.Duration(rand.Intn(1000)) * time.Millisecond
	}

	// Construct the routing table where the probe function simply looks up from
	// the latency table.
	//
	// We want to set up the sample period so that the routing table balances
	// the rings faster.
	config := NewDefaultRingsConfig(func(pid peer.ID) (time.Duration, error) {
		return latencyTable[pid], nil
	})
	config.SamplePeriod = time.Millisecond
	table := NewRingsRoutingTable(config)

	// Add nodes to the routing table
	for _, pid := range pids {
		table.Add(pid)
	}

	// Wait for the rings to balance
	time.Sleep(10 * time.Millisecond)

	// Recommend one node per ring
	recommended := table.Recommend(config.RingsCount, nil)
	if len(recommended) != config.RingsCount {
		t.Fatalf("Expected %v recommendations, got %v", config.RingsCount, len(recommended))
	}

	// Check the latency growth ratio
	var ratioTotal float64
	for i := 1; i < len(recommended); i++ {
		ratioTotal += float64(latencyTable[recommended[i]] / latencyTable[recommended[i-1]])
	}
	averageRatio := ratioTotal / float64(len(recommended)-1)
	if averageRatio < config.LatencyGrowthFactor-1 || averageRatio >
		config.LatencyGrowthFactor+1 {
		t.Fatalf("Incorrect latency growth ratio: %v", averageRatio)
	}
}
