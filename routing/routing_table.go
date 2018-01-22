/**
 * File        : routing_table.go
 * Description : High-level routing table interface.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package routing

import (
	"math"
	"math/rand"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmSAFA8v42u4gpJNy1tb7vW3JiiXiaYDC2b845c2RnNSJL/go-libp2p-kbucket"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/streamstore"
)

type RoutingTable interface {

	Add(remote peer.ID)

	Remove(remote peer.ID)

	Recommend() (candidates []peer.ID)

	Refer(remote peer.ID) (recommend bool, replce bool, replacee peer.ID)

	Sample(size int, exclude []peer.ID) (sample []peerstore.PeerInfo)

	Size() int
}

type kadlike struct {
	kademlia    *kbucket.RoutingTable
	local       peer.ID
	peerstore   peerstore.Peerstore
	streamstore streamstore.Streamstore
}

func NewRoutingTable(
	kbucketSize int,
	latency time.Duration,
	local peer.ID,
	peerstore peerstore.Peerstore,
	streamstore streamstore.Streamstore,
) RoutingTable {

	table := &kadlike{}
	table.kademlia = kbucket.NewRoutingTable(
		kbucketSize,
		kbucket.ConvertPeerID(local),
		latency,
		peerstore,
	)
	table.local = local
	table.peerstore = peerstore
	table.streamstore = streamstore

	return table

}

func (table *kadlike) Add(remote peer.ID) {
	table.kademlia.Update(remote)
}

func (table *kadlike) Remove(remote peer.ID) {
	table.kademlia.Remove(remote)
}

func (table *kadlike) Recommend() (candidates []peer.ID) {

	var delta []peer.ID
	var peers []peer.ID

	buckets := table.kademlia.Buckets
	streams := table.streamstore.Peers()
	targets := deal(table.streamstore.Capacity(), len(buckets))

	for i := range buckets {

		peers = buckets[i].Peers()
		delta = kbucket.SortClosestPeers(
			difference(peers, streams),
			kbucket.ConvertPeerID(table.local),
		)

		s := len(streams) - len(difference(streams, peers))
		t := targets[i]

		for len(delta) > 0 {

			if s >= t {
				break
			}

			n := float64(len(delta))
			j := int(math.Floor(math.Exp(math.Log(n+1)*rand.Float64()) - 1))

			info := table.peerstore.PeerInfo(delta[j])
			if info.ID != table.local && len(info.Addrs) != 0 {
				candidates = append(candidates, info.ID)
				t--
			}

			delta = append(delta[:j], delta[j+1:]...)

		}

	}

	return

}

func (table *kadlike) Refer(remote peer.ID) (recommend bool, replce bool, replacee peer.ID) {

	buckets := table.kademlia.Buckets
	streams := table.streamstore.Peers()
	targets := deal(table.streamstore.Capacity(), len(buckets))

	for i := range buckets {

		if buckets[i].Has(remote) {

			for j := 0; j < len(streams); j++ {
				if !buckets[i].Has(streams[j]) {
					copy(streams[j:], streams[j+1:])
					streams = streams[:len(streams)-1]
					j--
				}
			}

			if len(streams)+1 > targets[i] {

				overflow := kbucket.SortClosestPeers(
					append(streams, remote),
					kbucket.ConvertPeerID(table.local),
				)[targets[i]:]

				for k := range overflow {
					if overflow[k] == remote {
						return
					}
				}

				return true, true, overflow[len(overflow)-1]

			}

			return true, false, ""

		}

	}

	return

}

func (table *kadlike) Sample(n int, exclude []peer.ID) (sample []peerstore.PeerInfo) {

	peers := table.kademlia.ListPeers()

	for len(peers) > 0 {

		if n <= len(sample) {
			break
		}

		j := rand.Intn(len(peers))

		exists := false
		for i := range exclude {
			if peers[j] == exclude[i] {
				exists = true
				break
			}
		}

		if !exists {
			info := table.peerstore.PeerInfo(peers[j])
			if info.ID != table.local && len(info.Addrs) != 0 {
				sample = append(sample, info)
			}
		}

		peers = append(peers[:j], peers[j+1:]...)

	}

	return


}

func (table *kadlike) Size() int {
	return table.kademlia.Size()
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
