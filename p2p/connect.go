/**
 * File        : connect.go
 * Description : Stream discovery module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"math/rand"
	"time"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// Discover streams.
func (client *client) discoverStreams() func() {

	// Create a shutdown function.
	notify := make(chan struct{})
	shutdown := func() {
		close(notify)
	}

	// Replenish the stream store.
	go func() {
		for {
			select {
			case <-notify:
				return
			case <-time.After(1 * time.Second):
				client.replenishStreamstore()
			}
		}
	}()

	// Return the shutdown function.
	return shutdown

}

// Replenish the stream store, i.e. fill outbound streams to maximum capacity.
func (client *client) replenishStreamstore() {

	// We need to pair with this many more peers.
	need := client.streamstore.OutboundCapacity() - client.streamstore.OutboundSize()
	if need <= 0 {
		return
	}

	// A set of peers that we are already connected with.
	connectedPeers := make(map[peer.ID]bool)
	for _, pid := range append(client.streamstore.InboundPeers(),
		client.streamstore.OutboundPeers()...) {
		connectedPeers[pid] = true
	}

	// Iterate through the known peers randomly
	knownPeers := client.table.ListPeers()
	perm := rand.Perm(len(knownPeers))

	for i := 0; i < len(knownPeers) && need > 0; i++ {
		pid := knownPeers[perm[i]]
		// If we are not already connected with it, connect with it.
		if !connectedPeers[pid] {
			client.pair(pid)
			need--
		}
	}

}

// Divide a by b and split the remainder.
func deal(a, b int) []int {

	if b < 1 {
		return nil
	}

	q := a / b
	r := a % b

	result := make([]int, b)

	for i := 0; i < b; i++ {
		result[i] = q
	}

	for i := 0; i < r; i++ {
		result[i]++
	}

	return result

}

// Calculate the relative complement of y in x.
func difference(x, y []peer.ID) []peer.ID {

	var exists bool
	var result []peer.ID

	for i := 0; i < len(x); i++ {
		exists = false
		for j := 0; j < len(y); j++ {
			if x[i] == y[j] {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, x[i])
		}
	}

	return result

}
