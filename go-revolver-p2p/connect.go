/**
 * File        : connect.go
 * Description : Stream discovery module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"math"
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
				if client.streamstore.Size() < client.streamstore.Capacity() {
					client.replenishStreamstore()
				}
			}
		}
	}()

	// Return the shutdown function.
	return shutdown

}

// Replenish the stream store.
func (client *client) replenishStreamstore() {

	// Calculate the targets.
	targetE := client.streamstore.Capacity() / 2
	targetW := client.streamstore.Capacity() - targetE

	targetNE := targetE / 2
	targetSE := targetE - targetNE
	targetSW := targetW / 2
	targetNW := targetW - targetSW

	//
	queueNE, queueSE, queueSW, queueNW := neighbors(
		client.id,
		client.table.ListPeers(),
	)

	buckets := [][]peer.ID{queueNE, queueSE, queueSW, queueNW}
	targets := []int{targetNE, targetSE, targetSW, targetNW}
	streams := client.streamstore.Peers()

	for i := range buckets {

		peers := difference(buckets[i], streams)

		s := len(streams) - len(difference(streams, buckets[i]))
		t := targets[i]

		// Select candidates from this bucket.
		for len(peers) > 0 {
			if s >= t {
				break
			}
			n := float64(len(peers))
			j := int(math.Floor(math.Exp(math.Log(n+1)*rand.Float64()) - 1))
			info := client.peerstore.PeerInfo(peers[j])
			if info.ID != client.id && len(info.Addrs) != 0 {
				client.pair(info.ID)
				t--
			}
			peers = append(peers[:j], peers[j+1:]...)
		}

	}

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
