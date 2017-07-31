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
	"time"

	"github.com/libp2p/go-libp2p-peer"
)

// Discover streams.
func (client *client) discoverStreams() func() {

	// Create a shutdown function.
	notify := make(chan struct {}, 1)
	shutdown := func() {
		notify <-struct {}{}
	}

	// Replenish the stream store.
	go func() {
		rate := math.Log(120) / 30
		then := time.Now()
		Discovery:
		for {
			select {
			case <-notify:
				break Discovery
			default:
				if time.Since(then) < 30 * time.Second {
					time.Sleep(time.Second * time.Duration(math.Exp(rate * time.Since(then).Seconds())))
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
	targetL := client.streamstore.Capacity() / 2
	targetR := client.streamstore.Capacity() - targetL

	// Identify candidates for pairing.
	queueL, queueR := client.neighbors(client.table.ListPeers())
	if len(queueL) > targetL {
		queueL = queueL[:targetL]
	}
	if len(queueR) > targetR {
		queueR = queueR[:targetR]
	}
	candidates := difference(
		append(queueL, queueR...),
		client.streamstore.Peers(),
	)

	// Pair with candidates.
	for i := range candidates {

		if client.streamstore.Size() == client.streamstore.Capacity() {
			break
		}

		info := client.peerstore.PeerInfo(candidates[i])
		if len(info.Addrs) == 0 {
			continue
		}

		_, err := client.pair(candidates[i])
		if err != nil {
			continue
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
