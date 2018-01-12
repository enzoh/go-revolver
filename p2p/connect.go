/**
 * File        : connect.go
 * Description : Stream discovery module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"math"
	"math/rand"
	"time"

	"gx/ipfs/QmSAFA8v42u4gpJNy1tb7vW3JiiXiaYDC2b845c2RnNSJL/go-libp2p-kbucket"
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
		rate := math.Log(120) / 30
		then := time.Now()
		for {
			select {
			case <-notify:
				return
			default:
				if time.Since(then) < 30*time.Second {
					time.Sleep(time.Second * time.Duration(math.Exp(rate*time.Since(then).Seconds())))
				} else {
					time.Sleep(120 * time.Second)
				}
				if client.streamstore.OutgoingSize() < client.streamstore.OutgoingCapacity() {
					client.replenishStreamstore()
				}
			}
		}
	}()

	// Return the shutdown function.
	return shutdown

}

// Replenish the stream store, i.e. fill outgoing streams to maximum capacity.
func (client *client) replenishStreamstore() {

	var delta []peer.ID
	var peers []peer.ID

	buckets := client.table.Buckets
	streams := client.streamstore.OutgoingPeers()
	targets := deal(client.streamstore.OutgoingCapacity(), len(buckets))

	for i := range buckets {

		peers = buckets[i].Peers()
		delta = kbucket.SortClosestPeers(
			difference(peers, streams),
			kbucket.ConvertPeerID(client.id),
		)

		s := len(streams) - len(difference(streams, peers))
		t := targets[i]

		// Select candidates from this bucket.
		for len(delta) > 0 {
			if s >= t {
				break
			}
			n := float64(len(delta))
			j := int(math.Floor(math.Exp(math.Log(n+1)*rand.Float64()) - 1))
			info := client.peerstore.PeerInfo(delta[j])
			if info.ID != client.id && len(info.Addrs) != 0 {
				client.pair(info.ID)
				t--
			}
			delta = append(delta[:j], delta[j+1:]...)
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
