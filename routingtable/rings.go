package routingtable

import (
	"math/rand"
	"sync"
	"time"

	"github.com/enzoh/go-logging"
	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// RingsConfig configures a Ring-based routing table
type RingsConfig struct {
	RingsPerNode        int
	NodesPerRing        int
	BaseLatency         float64
	LatencyGrowthFactor float64
	SampleSize          int
	SamplePeriod        time.Duration
	// A function for retrieving the up-to-date latency information for a given
	// peer.
	LatencyProbFn func(peer.ID) (time.Duration, error)

	Logger logging.Logger
}

// ringsRoutingTable is a RoutingTable based on latency rings.
type ringsRoutingTable struct {
	sync.RWMutex

	// The config
	conf RingsConfig

	// A set of known peers
	peers map[peer.ID]bool

	// The rings
	rings []*ring

	// The latency range of the rings.  Specifically, for the nth ring, the
	// latency range is [latRanges[n], latRanges[n+1]).
	latRanges []float64

	// For storing latency info
	metrics peerstore.Metrics

	// Shutdown signal
	shutdown chan struct{}
}

// A ring stores a list of peers within a certain latency range
type ring struct {
	peers []peer.ID
}

// Add a peer to the ring
func (r *ring) Add(pid peer.ID) {
	r.peers = append(r.peers, pid)
}

// Remove a peer from the ring
func (r *ring) Remove(pid peer.ID) {
	for i, peer := range r.peers {
		if peer == pid {
			r.peers = append(r.peers[:i], r.peers[i+1:]...)
		}
	}
}

// Return k random peers in the ring.  If a peer is in the preferred list,
// return it.
func (r *ring) Recommend(k int, preferred []peer.ID) []peer.ID {
	recommended := make(map[peer.ID]bool)

	// Recommend preferred peers if possible
	for _, pid := range r.peers {
		for _, pref := range preferred {
			if pid == pref {
				recommended[pid] = true
			}
		}
	}

	// Fill the rest of the recommendation with a random sample of the peers
	for _, i := range rand.Perm(len(r.peers)) {
		if len(recommended) >= k {
			break
		}
		recommended[r.peers[i]] = true
	}

	// Return a list
	var res []peer.ID
	for pid := range recommended {
		res = append(res, pid)
	}

	return res
}

// NewDefaultRingsRoutingTable creates a RoutingTable based on rings using
// default parameters.
func NewDefaultRingsRoutingTable() RoutingTable {
	return NewRingsRoutingTable(RingsConfig{
		RingsPerNode:        9,
		NodesPerRing:        16,
		BaseLatency:         2,
		LatencyGrowthFactor: 2,
		SampleSize:          16,
		SamplePeriod:        30 * time.Second,
	})
}

// NewRingsRoutingTable creates a RoutingTable with the given config.
func NewRingsRoutingTable(conf RingsConfig) RoutingTable {
	// Construct the latency ranges
	// The first element is always going to be 0.
	latRanges := []float64{0}
	k := conf.BaseLatency
	for i := 1; i < conf.RingsPerNode; i++ {
		latRanges = append(latRanges, k)
		k *= conf.LatencyGrowthFactor
	}

	r := &ringsRoutingTable{
		conf:      conf,
		peers:     make(map[peer.ID]bool),
		metrics:   peerstore.NewMetrics(),
		latRanges: latRanges,
		shutdown:  make(chan struct{}),
	}

	// Periodically refresh latency and re-balance rings until explicitly shut
	// down.
	go func() {
		select {
		case <-time.After(r.conf.SamplePeriod):
			r.refreshLatency()
			r.populateRings()
		case <-r.shutdown:
			return
		}
	}()

	return r
}

// refreshLatency picks a random subset of peers and refresh their latency
// information.  The rings are then re-populated.
func (r *ringsRoutingTable) refreshLatency() {
	// Get a list of all peers
	var pids []peer.ID
	for pid := range r.peers {
		pids = append(pids, pid)
	}
	peerCount := len(pids)

	// Get a random sample of the peers
	var sample []peer.ID
	perm := rand.Perm(peerCount)
	for i := 0; i < r.conf.SampleSize && i < peerCount; i++ {
		sample = append(sample, pids[perm[i]])
	}

	for _, pid := range sample {
		latency, err := r.conf.LatencyProbFn(pid)
		if err != nil {
			r.conf.Logger.Errorf("error probing latency of peer %v", pid)
		} else {
			func() {
				r.Lock()
				defer r.Unlock()
				r.metrics.RecordLatency(pid, latency)
			}()
		}
	}
}

// populateRings puts peers into the rings.
func (r *ringsRoutingTable) populateRings() {
	r.Lock()
	defer r.Unlock()

	var rings []*ring
	for i := 0; i < r.conf.RingsPerNode; i++ {
		rings = append(rings, &ring{})
	}

	for pid := range r.peers {
		latency := r.metrics.LatencyEWMA(pid)
		// Find the ring that the peer belongs to
		for i := len(r.latRanges) - 1; i >= 0; i-- {
			if latency > time.Duration(r.latRanges[i]*float64(time.Millisecond)) {
				r.rings[i].Add(pid)
			}
		}
	}

	r.rings = rings
}

func (r *ringsRoutingTable) Add(pid peer.ID) {
	// Do nothing if we already know about this peer.
	if func() bool {
		r.RLock()
		defer r.RUnlock()
		return r.peers[pid]
	}() {
		return
	}

	// Otherwise, ping it and record latency info.
	// Note how we don't want to hold the lock while pinging it.
	latency, err := r.conf.LatencyProbFn(pid)
	if err != nil {
		r.conf.Logger.Errorf("Error probing peer %s", pid)
		return
	}

	r.Lock()
	defer r.Unlock()

	r.metrics.RecordLatency(pid, latency)
	r.peers[pid] = true
}

func (r *ringsRoutingTable) Remove(pid peer.ID) {
	r.Lock()
	defer r.Unlock()
	delete(r.peers, pid)
	for _, ring := range r.rings {
		ring.Remove(pid)
	}
	// TODO: remove the peer from the metrics store too
}

func (r *ringsRoutingTable) Recommend(count int, preferred []peer.ID) []peer.ID {
	// Compute how many nodes we want from each ring
	nodesFromRing := make([]int, r.conf.RingsPerNode)

	// TODO: if count is less than the number of rings, we actually want to
	// select from rings that are evenly spaced out.  For instance, if count is
	// 3 and we have 9 rings, then we want to select from the 0th, the 3rd, and
	// the 6th ring.
	var j int // index for ring
	for i := 0; i < count; i++ {
		nodesFromRing[j]++
		j++
		if j >= r.conf.RingsPerNode {
			// Reset index
			j = 0
		}
	}

	var recommended []peer.ID
	for i, count := range nodesFromRing {
		recommended = append(recommended, r.rings[i].Recommend(count, preferred)...)
	}
	return recommended
}

// TODO
func (r *ringsRoutingTable) Sample() []peer.ID {
	return nil
}

func (r *ringsRoutingTable) Size() int {
	return len(r.peers)
}

func (r *ringsRoutingTable) Shutdown() {
	close(r.shutdown)
}
