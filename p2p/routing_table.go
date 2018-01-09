/**
 * File        : routing_table.go
 * Description : System for organizing peers and prioritizing streams.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"crypto/sha256"
	"sort"
	"sync"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

type RoutingTable interface {

	// Add a peer to a routing table.
	Add(remote peer.ID)

	// Remove a peer from a routing table.
	Remove(remote peer.ID)

	// Get a recommended slice of peers from a routing table.
	Recommend(int) []peer.ID

	// Get a random slice of peers from a routing table.
	Random(int) []peer.ID

	// Get the size of a routing table.
	Size() int
}

type torus struct {
	local peer.ID
	cache map[peer.ID]int
	xs    []*xref
	ts    []*tref
	peerstore.Metrics
	sync.RWMutex
}

type xref struct {
	x   []byte
	ref peer.ID
}

type tref struct {
	t   time.Duration
	ref peer.ID
}

// Create a routing table.
func NewRoutingTable(local peer.ID, metrics peerstore.Metrics) RoutingTable {
	return &torus{local, make(map[peer.ID]int), nil, nil, metrics, sync.RWMutex{}}
}

// Add a peer to a routing table.
func (torus *torus) Add(remote peer.ID) {
	torus.Lock()
	defer torus.Unlock()
	h := sha256.Sum256([]byte(remote))
	x := &xref{h[:], remote}
	i := sort.Search(len(torus.xs), func(i int) bool {
		return bytes.Compare(torus.xs[i].x, x.x) >= 0
	})
	if !(i < len(torus.xs) && torus.xs[i].ref == x.ref) {
		torus.xs = append(torus.xs, x)
		copy(torus.xs[i+1:], torus.xs[i:])
		torus.xs[i] = x
	}
	t := &tref{torus.LatencyEWMA(remote), remote}
	j, ok := torus.cache[t.ref]
	if ok {
		torus.ts = append(torus.ts[:j], torus.ts[j+1:]...)
	}
	k := sort.Search(len(torus.ts), func(k int) bool {
		return torus.ts[k].t > t.t
	})
	torus.ts = append(torus.ts, t)
	copy(torus.ts[k+1:], torus.ts[k:])
	torus.ts[k] = t
	torus.cache[t.ref] = k
}

// Remove a peer from a routing table.
func (table *torus) Remove(remote peer.ID) {
}

func (table *torus) Recommend(n int) []peer.ID {
	return nil
}

func (table *torus) Random(n int) []peer.ID {
	return nil
}

// Get the size of the routing table.
func (table *torus) Size() int {
	return len(table.xs)
}
